package main

import (
	"fmt"
	"log"
	stdnet "net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func sanitizeDomain(domain string) (string, error) {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" || !domainNamePattern.MatchString(domain) || strings.Contains(domain, "..") {
		return "", fmt.Errorf("invalid domain")
	}
	return domain, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func reloadNginx() error {
	cmd := exec.Command("systemctl", "reload", "nginx")
	if output, err := cmd.CombinedOutput(); err == nil {
		return nil
	} else if fallbackOutput, fallbackErr := exec.Command("nginx", "-s", "reload").CombinedOutput(); fallbackErr != nil {
		return fmt.Errorf("systemctl reload nginx failed: %s | nginx -s reload failed: %s", strings.TrimSpace(string(output)), strings.TrimSpace(string(fallbackOutput)))
	}
	return nil
}

func ensureLogFileExists(path string) {
	if runtime.GOOS == "windows" {
		dir := filepath.Dir(path)
		_ = os.MkdirAll(dir, 0755)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			_ = os.WriteFile(path, []byte(""), 0644)
		}
		return
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = os.WriteFile(path, []byte(""), 0640)
		_ = exec.Command("chown", "www-data:adm", path).Run()
		_ = exec.Command("chmod", "640", path).Run()
	}
}

func runSQLCommand(sql string) (string, error) {
	config, err := loadDBConfig()
	if err != nil {
		return "", err
	}
	if config.Host == "" {
		return "", fmt.Errorf("Database connection is not configured")
	}

	args := []string{"-h", config.Host, "-P", config.Port, "-u", config.Username, "-e", sql}
	cmd := exec.Command("mysql", args...)
	cmd.Env = os.Environ()
	if config.Password != "" {
		cmd.Env = append(cmd.Env, "MYSQL_PWD="+config.Password)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err.Error(), strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func executeCustomSQL(dbName string, query string) (gin.H, error) {
	db, err := getDBConnection(dbName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	trimmed := strings.TrimSpace(strings.ToLower(query))
	isSelect := strings.HasPrefix(trimmed, "select") ||
		strings.HasPrefix(trimmed, "show") ||
		strings.HasPrefix(trimmed, "desc") ||
		strings.HasPrefix(trimmed, "explain") ||
		strings.HasPrefix(trimmed, "help")

	if isSelect {
		sqlRows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer sqlRows.Close()

		columns, err := sqlRows.Columns()
		if err != nil {
			return nil, err
		}

		var rows [][]interface{}
		for sqlRows.Next() {
			rowValues := make([]interface{}, len(columns))
			rowValPointers := make([]interface{}, len(columns))
			for i := range rowValues {
				rowValPointers[i] = &rowValues[i]
			}

			if err := sqlRows.Scan(rowValPointers...); err != nil {
				return nil, err
			}

			formattedRow := make([]interface{}, len(columns))
			for i, val := range rowValues {
				if val == nil {
					formattedRow[i] = nil
				} else if b, ok := val.([]byte); ok {
					formattedRow[i] = string(b)
				} else {
					formattedRow[i] = val
				}
			}
			rows = append(rows, formattedRow)
		}

		if rows == nil {
			rows = [][]interface{}{}
		}

		return gin.H{
			"type":    "select",
			"columns": columns,
			"rows":    rows,
		}, nil
	} else {
		res, err := db.Exec(query)
		if err != nil {
			return nil, err
		}

		rowsAffected, _ := res.RowsAffected()
		lastInsertID, _ := res.LastInsertId()

		return gin.H{
			"type":           "exec",
			"rows_affected":  rowsAffected,
			"last_insert_id": lastInsertID,
		}, nil
	}
}

func generateRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	bytes := make([]byte, n)
	seed := time.Now().UnixNano()
	for i := 0; i < n; i++ {
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		bytes[i] = letters[seed%int64(len(letters))]
	}
	return string(bytes)
}

func isPrivateOrLocalIP(ipStr string) bool {
	ip := stdnet.ParseIP(ipStr)
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7",
	}
	for _, block := range privateBlocks {
		_, subnet, err := stdnet.ParseCIDR(block)
		if err == nil && subnet.Contains(ip) {
			return true
		}
	}
	return false
}

func sendTelegram(message string) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chatID == "" {
		return
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	formData := url.Values{
		"chat_id": {chatID},
		"text":    {message},
	}

	resp, err := http.PostForm(apiURL, formData)
	if err != nil {
		log.Printf("[TELEGRAM ERROR] Failed to send alert: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[TELEGRAM ERROR] Telegram API returned status %d\n", resp.StatusCode)
	}
}
