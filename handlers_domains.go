package main

import (
	"bufio"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func registerDomainRoutes(api *gin.RouterGroup) {
	api.GET("/domains", func(c *gin.Context) {
		scan := c.Query("scan") == "true"
		c.JSON(200, getDomains(scan))
	})

	api.POST("/domains/star", func(c *gin.Context) {
		var req struct {
			Domain  string `json:"domain"`
			Starred bool   `json:"starred"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		domain, err := sanitizeDomain(req.Domain)
		if err != nil {
			c.JSON(400, gin.H{"error": "Domain is not allowed"})
			return
		}

		if err := updateDomainStar(domain, req.Starred); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"status":  "ok",
			"domain":  domain,
			"starred": req.Starred,
		})
	})

	api.GET("/domains/config", func(c *gin.Context) {
		domain := c.Query("domain")
		if domain == "" {
			c.JSON(400, gin.H{"error": "Domain is required"})
			return
		}
		domain, err := sanitizeDomain(domain)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid domain"})
			return
		}

		configPath, err := findDomainConfigPath(domain)
		if err != nil {
			c.JSON(404, gin.H{"error": "Configuration not found for domain"})
			return
		}

		content, err := os.ReadFile(configPath)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to read config file: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"domain":  domain,
			"content": string(content),
			"path":    configPath,
		})
	})

	api.POST("/domains/config", func(c *gin.Context) {
		var req struct {
			Domain  string `json:"domain"`
			Content string `json:"content"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		domain, err := sanitizeDomain(req.Domain)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid domain"})
			return
		}

		configPath, err := findDomainConfigPath(domain)
		if err != nil {
			c.JSON(404, gin.H{"error": "Configuration not found for domain"})
			return
		}

		if runtime.GOOS == "windows" {
			if err := os.WriteFile(configPath, []byte(req.Content), 0644); err != nil {
				c.JSON(500, gin.H{"error": "Failed to write config file: " + err.Error()})
				return
			}
			c.JSON(200, gin.H{"status": "ok"})
			return
		}

		// Under Linux:
		oldContent, err := os.ReadFile(configPath)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to read existing config: " + err.Error()})
			return
		}

		if err := os.WriteFile(configPath, []byte(req.Content), 0644); err != nil {
			c.JSON(500, gin.H{"error": "Failed to write config file: " + err.Error()})
			return
		}

		cmd := exec.Command("nginx", "-t")
		output, err := cmd.CombinedOutput()
		if err != nil {
			_ = os.WriteFile(configPath, oldContent, 0644)
			c.JSON(400, gin.H{"error": "Invalid Nginx configuration:\n" + string(output)})
			return
		}

		if err := reloadNginx(); err != nil {
			_ = os.WriteFile(configPath, oldContent, 0644)
			_ = reloadNginx()
			c.JSON(500, gin.H{"error": "Failed to reload Nginx: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "ok"})
	})

	api.POST("/domains/delete", func(c *gin.Context) {
		var req struct {
			Domain     string `json:"domain"`
			DeleteDB   bool   `json:"delete_db"`
			DeleteRoot bool   `json:"delete_root"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		domain, err := sanitizeDomain(req.Domain)
		if err != nil {
			c.JSON(400, gin.H{"error": "Domain is not allowed"})
			return
		}

		result, err := deleteDomain(domain, req.DeleteDB, req.DeleteRoot)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"status":  "ok",
			"message": fmt.Sprintf("Deleted domain %s", domain),
			"result":  result,
		})
	})

	api.POST("/domains/create", func(c *gin.Context) {
		var req struct {
			Domain      string `json:"domain"`
			Type        string `json:"type"`          // "static", "php", "proxy"
			PHPVersion  string `json:"php_version"`   // "8.3", "7.4"
			ProxyPass   string `json:"proxy_pass"`    // e.g. http://127.0.0.1:8080
			CreateDB    bool   `json:"create_db"`
			SSL         bool   `json:"ssl"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		domain, err := sanitizeDomain(req.Domain)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid domain format"})
			return
		}

		paths := getDomainPaths()
		availablePath := filepath.Join(paths.sitesAvailableDir, domain+".conf")
		enabledPath := filepath.Join(paths.sitesEnabledDir, domain+".conf")

		if fileExists(availablePath) || fileExists(enabledPath) {
			c.JSON(400, gin.H{"error": "Website / Domain configuration already exists"})
			return
		}

		var noteContent []string

		if req.Type == "static" || req.Type == "php" {
			webRoot := filepath.Join("/home", domain)
			if runtime.GOOS == "windows" {
				webRoot = filepath.Join(".", "logs", "www", domain)
			}

			if err := os.MkdirAll(webRoot, 0755); err != nil {
				c.JSON(500, gin.H{"error": "Failed to create web root directory: " + err.Error()})
				return
			}

			if req.Type == "static" {
				indexPath := filepath.Join(webRoot, "index.html")
				if !fileExists(indexPath) {
					defaultHTML := fmt.Sprintf("<!DOCTYPE html><html><head><title>Welcome to %s</title></head><body style='font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #0f172a; color: #f1f5f9;'><div><h1>%s has been successfully configured!</h1><p>Website is running under Static HTML mode.</p></div></body></html>", domain, domain)
					_ = os.WriteFile(indexPath, []byte(defaultHTML), 0644)
				}
			} else if req.Type == "php" {
				indexPath := filepath.Join(webRoot, "index.php")
				if !fileExists(indexPath) {
					defaultPHP := fmt.Sprintf("<?php\necho '<h1>Welcome to %s</h1>';\necho '<p>Website is running under PHP Mode (PHP Version: %s)</p>';\nphpinfo();", domain, req.PHPVersion)
					_ = os.WriteFile(indexPath, []byte(defaultPHP), 0644)
				}
			}
		}

		// Handle Database Provisioning
		var dbName, dbUser, dbPass string
		if req.CreateDB {
			prefix := strings.ReplaceAll(domain, ".", "_")
			if len(prefix) > 10 {
				prefix = prefix[:10]
			}
			var dbErr error
			dbName, dbUser, dbPass, dbErr = provisionCMSDatabase(prefix)
			if dbErr != nil {
				c.JSON(500, gin.H{"error": "Database provisioning failed: " + dbErr.Error()})
				return
			}
			noteContent = append(noteContent, fmt.Sprintf("Database: %s\nUser: %s\nPass: %s", dbName, dbUser, dbPass))
		}

		// Generate Nginx Configuration
		var nginxConfig string
		if req.Type == "static" {
			webRoot := filepath.Join("/home", domain)
			accLogPath := filepath.ToSlash(filepath.Join(paths.nginxLogDir, domain+"_access.log"))
			errLogPath := filepath.ToSlash(filepath.Join(paths.nginxLogDir, domain+"_error.log"))
			nginxConfig = fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    root %s;
    index index.html index.htm;

    access_log %s;
    error_log %s;

    location ~ /\.(env|git|ht|svn) {
        deny all;
    }

    location / {
        try_files $uri $uri/ =404;
    }
}
`, domain, webRoot, accLogPath, errLogPath)
		} else if req.Type == "php" {
			webRoot := filepath.Join("/home", domain)
			sockPath := "/run/php/php8.3-fpm.sock"
			if req.PHPVersion == "7.4" {
				sockPath = "/run/php/php7.4-fpm.sock"
			}
			accLogPath := filepath.ToSlash(filepath.Join(paths.nginxLogDir, domain+"_access.log"))
			errLogPath := filepath.ToSlash(filepath.Join(paths.nginxLogDir, domain+"_error.log"))
			nginxConfig = fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    root %s;
    index index.php index.html index.htm;

    access_log %s;
    error_log %s;

    location ~ /\.(env|git|ht|svn) {
        deny all;
    }

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:%s;
    }
}
`, domain, webRoot, accLogPath, errLogPath, sockPath)
		} else if req.Type == "proxy" {
			accLogPath := filepath.ToSlash(filepath.Join(paths.nginxLogDir, domain+"_access.log"))
			errLogPath := filepath.ToSlash(filepath.Join(paths.nginxLogDir, domain+"_error.log"))
			nginxConfig = fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    access_log %s;
    error_log %s;

    location ~ /\.(env|git|ht|svn) {
        deny all;
    }

    location / {
        proxy_pass %s;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
`, domain, accLogPath, errLogPath, req.ProxyPass)
		} else {
			c.JSON(400, gin.H{"error": "Invalid website type"})
			return
		}

		if err := os.MkdirAll(filepath.Dir(availablePath), 0755); err != nil {
			c.JSON(500, gin.H{"error": "Failed to create nginx config directory: " + err.Error()})
			return
		}
		if err := os.WriteFile(availablePath, []byte(nginxConfig), 0644); err != nil {
			c.JSON(500, gin.H{"error": "Failed to write nginx available config: " + err.Error()})
			return
		}

		if err := os.MkdirAll(filepath.Dir(enabledPath), 0755); err != nil {
			c.JSON(500, gin.H{"error": "Failed to create nginx enabled directory: " + err.Error()})
			return
		}
		_ = os.Remove(enabledPath)
		if err := os.Symlink(availablePath, enabledPath); err != nil {
			if runtime.GOOS != "windows" {
				c.JSON(500, gin.H{"error": "Failed to symlink nginx configuration: " + err.Error()})
				return
			}
		}

		if len(noteContent) > 0 {
			noteStr := strings.Join(noteContent, "\n")
			_ = updateDomainNote(domain, noteStr)
		}

		accLogPath := filepath.Join(paths.nginxLogDir, domain+"_access.log")
		errLogPath := filepath.Join(paths.nginxLogDir, domain+"_error.log")
		ensureLogFileExists(accLogPath)
		ensureLogFileExists(errLogPath)

		if runtime.GOOS != "windows" {
			if err := reloadNginx(); err != nil {
				c.JSON(500, gin.H{"error": "Failed to reload Nginx: " + err.Error()})
				return
			}
			if req.SSL {
				runCertbot(domain)
			}
		}

		clearDomainCache()

		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Website created successfully",
		})
	})

	api.POST("/domains/note", func(c *gin.Context) {
		var req struct {
			Domain string `json:"domain"`
			Note   string `json:"note"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		domain, err := sanitizeDomain(req.Domain)
		if err != nil {
			c.JSON(400, gin.H{"error": "Domain is not allowed"})
			return
		}

		note := strings.TrimSpace(req.Note)
		if len(note) > 500 {
			c.JSON(400, gin.H{"error": "Note is too long"})
			return
		}

		if err := updateDomainNote(domain, note); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"status": "ok",
			"domain": domain,
			"note":   note,
		})
	})

	api.GET("/ssl", func(c *gin.Context) {
		c.JSON(200, getSSLCertificates())
	})

	api.POST("/ssl/renew", func(c *gin.Context) {
		var req struct {
			Domain string `json:"domain"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		domain, err := sanitizeDomain(req.Domain)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid domain"})
			return
		}

		if runtime.GOOS == "windows" {
			c.JSON(200, gin.H{"status": "ok", "message": "Simulation: Renewed SSL for " + domain})
			return
		}

		cmd := exec.Command("certbot", "renew", "--cert-name", domain, "--non-interactive")
		output, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to renew SSL: " + err.Error(), "details": string(output)})
			return
		}

		_ = reloadNginx()

		c.JSON(200, gin.H{"status": "ok", "message": string(output)})
	})
}

func getDomainPaths() domainPaths {
	paths := domainPaths{
		sitesEnabledDir:   "/etc/nginx/sites-enabled",
		sitesAvailableDir: "/etc/nginx/sites-available",
		nginxLogDir:       "/var/log/nginx",
	}
	if runtime.GOOS == "windows" {
		paths.sitesEnabledDir = "./logs/sites-enabled"
		paths.sitesAvailableDir = "./logs/sites-available"
		paths.nginxLogDir = "./logs/nginx"
	}
	return paths
}

func clearDomainCache() {
	cachedDomains = nil
	lastDomainCheck = time.Time{}
}

func getDomainNotesPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(".", "data", "domain-notes.json")
	}
	return filepath.Join("/usr/local/bin", "data", "domain-notes.json")
}

func loadDomainNotes() map[string]string {
	path := getDomainNotesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}

	notes := map[string]string{}
	if err := json.Unmarshal(data, &notes); err != nil {
		return map[string]string{}
	}
	return notes
}

func saveDomainNotes(notes map[string]string) error {
	path := getDomainNotesPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func updateDomainNote(domain string, note string) error {
	notes := loadDomainNotes()
	note = strings.TrimSpace(note)
	if note == "" {
		delete(notes, domain)
	} else {
		notes[domain] = note
	}

	if err := saveDomainNotes(notes); err != nil {
		return err
	}

	clearDomainCache()
	return nil
}

func getDomainStarsPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(".", "data", "domain-stars.json")
	}
	return filepath.Join("/usr/local/bin", "data", "domain-stars.json")
}

func loadDomainStars() map[string]bool {
	path := getDomainStarsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]bool{}
	}

	stars := map[string]bool{}
	if err := json.Unmarshal(data, &stars); err != nil {
		return map[string]bool{}
	}
	return stars
}

func saveDomainStars(stars map[string]bool) error {
	path := getDomainStarsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(stars, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func updateDomainStar(domain string, starred bool) error {
	stars := loadDomainStars()
	if !starred {
		delete(stars, domain)
	} else {
		stars[domain] = true
	}

	if err := saveDomainStars(stars); err != nil {
		return err
	}

	clearDomainCache()
	return nil
}

func getDomainConfigCandidates(domain string) []string {
	paths := getDomainPaths()
	return []string{
		filepath.Join(paths.sitesEnabledDir, domain),
		filepath.Join(paths.sitesEnabledDir, domain+".conf"),
		filepath.Join(paths.sitesAvailableDir, domain),
		filepath.Join(paths.sitesAvailableDir, domain+".conf"),
	}
}

func findDomainConfigPath(domain string) (string, error) {
	for _, path := range getDomainConfigCandidates(domain) {
		if fileExists(path) {
			return path, nil
		}
	}
	return "", fmt.Errorf("domain config not found")
}

func parseNginxRoot(configPath string) (string, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if !strings.HasPrefix(line, "root ") {
			continue
		}
		root := strings.TrimSpace(strings.TrimPrefix(line, "root"))
		root = strings.TrimSuffix(root, ";")
		root = strings.Trim(root, `"'`)
		if root != "" {
			return filepath.Clean(root), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("root directive not found")
}

func getAppRoot(rootPath string) string {
	cleanRoot := filepath.Clean(rootPath)
	if strings.EqualFold(filepath.Base(cleanRoot), "public") {
		return filepath.Dir(cleanRoot)
	}
	return cleanRoot
}

func isAllowedRootDeletePath(path string) bool {
	cleanPath := filepath.Clean(path)
	if cleanPath == "" || cleanPath == "." || cleanPath == string(filepath.Separator) {
		return false
	}

	allowedPrefixes := []string{
		filepath.Clean("/var/www"),
		filepath.Clean("/home"),
		filepath.Clean("/srv/www"),
		filepath.Clean("/opt"),
	}
	if runtime.GOOS == "windows" {
		allowedPrefixes = []string{
			filepath.Clean("./logs/www"),
			filepath.Clean("./www"),
		}
	}

	for _, prefix := range allowedPrefixes {
		if cleanPath == prefix || strings.HasPrefix(cleanPath, prefix+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		values[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func removeIfExists(path string) error {
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return err
}

func removeAllIfExists(path string) error {
	if !fileExists(path) {
		return nil
	}
	return os.RemoveAll(path)
}

func dropDatabaseFromEnv(appRoot string) (string, error) {
	envPath := filepath.Join(appRoot, ".env")
	envValues, err := parseEnvFile(envPath)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", envPath, err)
	}

	dbName := envValues["DB_DATABASE"]
	if dbName == "" {
		return "", fmt.Errorf("DB_DATABASE not found in %s", envPath)
	}
	if !dbNamePattern.MatchString(dbName) {
		return "", fmt.Errorf("database name is not allowed")
	}

	dbConn := strings.ToLower(envValues["DB_CONNECTION"])
	if dbConn != "" && dbConn != "mysql" && dbConn != "mariadb" {
		return "", fmt.Errorf("unsupported DB_CONNECTION: %s", dbConn)
	}

	dbUser := envValues["DB_USERNAME"]
	if dbUser == "" {
		return "", fmt.Errorf("DB_USERNAME not found in %s", envPath)
	}

	args := []string{"-u", dbUser}
	if host := envValues["DB_HOST"]; host != "" {
		args = append(args, "-h", host)
	}
	if port := envValues["DB_PORT"]; port != "" {
		args = append(args, "-P", port)
	}
	args = append(args, "-e", fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName))

	cmd := exec.Command("mysql", args...)
	cmd.Env = os.Environ()
	if password := envValues["DB_PASSWORD"]; password != "" {
		cmd.Env = append(cmd.Env, "MYSQL_PWD="+password)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("drop database failed: %s", strings.TrimSpace(string(output)))
	}

	return dbName, nil
}

func deleteDomain(domain string, deleteDB bool, deleteRoot bool) (domainDeleteResult, error) {
	result := domainDeleteResult{
		Domain:     domain,
		DeleteDB:   deleteDB,
		DeleteRoot: deleteRoot,
	}

	metaList, _ := loadAppsMetadata()
	var dockerApp *AppMetadata
	for _, m := range metaList {
		if m.Domain == domain {
			dockerApp = &m
			break
		}
	}

	if dockerApp != nil {
		_ = exec.Command("docker", "rm", "-f", dockerApp.ID).Run()
		_ = exec.Command("docker", "volume", "rm", dockerApp.ID+"_data").Run()
		result.Deleted = append(result.Deleted, "docker-container:"+dockerApp.ID)
		result.Deleted = append(result.Deleted, "docker-volume:"+dockerApp.ID+"_data")

		if deleteRoot {
			appRoot := filepath.Join("/home", domain)
			if runtime.GOOS == "windows" {
				appRoot = filepath.Join(".", "logs", "www", domain)
			}
			if isAllowedRootDeletePath(appRoot) {
				_ = os.RemoveAll(appRoot)
				result.Deleted = append(result.Deleted, "directory:"+appRoot)
			}
		}

		if dockerApp.DBName != "" {
			_, _ = runSQLCommand(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%';", dockerApp.DBUser))
			_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", dockerApp.DBName))
			_, _ = runSQLCommand("FLUSH PRIVILEGES;")
			result.Database = dockerApp.DBName
			result.Deleted = append(result.Deleted, "database:"+dockerApp.DBName)
		}

		for _, configPath := range getDomainConfigCandidates(domain) {
			existed := fileExists(configPath)
			if err := removeIfExists(configPath); err == nil && existed {
				result.Deleted = append(result.Deleted, configPath)
			}
		}

		paths := getDomainPaths()
		logFiles := []string{
			filepath.Join(paths.nginxLogDir, domain+"_access.log"),
			filepath.Join(paths.nginxLogDir, domain+"_error.log"),
		}
		for _, logPath := range logFiles {
			existed := fileExists(logPath)
			if err := removeIfExists(logPath); err == nil && existed {
				result.Deleted = append(result.Deleted, logPath)
			}
		}

		_ = updateDomainNote(domain, "")
		var remainingMeta []AppMetadata
		for _, item := range metaList {
			if item.Domain != domain {
				remainingMeta = append(remainingMeta, item)
			}
		}
		_ = saveAppsMetadata(remainingMeta)

		if runtime.GOOS != "windows" {
			_ = reloadNginx()
			result.NginxReload = true
		}
		clearDomainCache()
		return result, nil
	}

	configPath, err := findDomainConfigPath(domain)
	if err != nil {
		return result, err
	}

	rootPath, err := parseNginxRoot(configPath)
	if err == nil {
		result.RootPath = getAppRoot(rootPath)
	}

	if deleteRoot {
		if result.RootPath == "" {
			return result, fmt.Errorf("cannot detect root path from nginx config")
		}
		if !isAllowedRootDeletePath(result.RootPath) {
			return result, fmt.Errorf("root path is outside allowed delete scope: %s", result.RootPath)
		}
	}

	if deleteDB {
		notes := loadDomainNotes()
		note := notes[domain]
		var noteDBName, noteDBUser string
		for _, l := range strings.Split(note, "\n") {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "Database:") {
				noteDBName = strings.TrimSpace(strings.TrimPrefix(l, "Database:"))
			} else if strings.HasPrefix(l, "User:") {
				noteDBUser = strings.TrimSpace(strings.TrimPrefix(l, "User:"))
			}
		}

		if noteDBName != "" {
			if dbNamePattern.MatchString(noteDBName) {
				if noteDBUser != "" {
					_, _ = runSQLCommand(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%';", noteDBUser))
				}
				_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", noteDBName))
				_, _ = runSQLCommand("FLUSH PRIVILEGES;")
				result.Database = noteDBName
				result.Deleted = append(result.Deleted, "database:"+noteDBName)
			}
		} else {
			if result.RootPath != "" {
				envPath := filepath.Join(result.RootPath, ".env")
				if fileExists(envPath) {
					dbName, dbErr := dropDatabaseFromEnv(result.RootPath)
					if dbErr != nil {
						return result, dbErr
					}
					result.Database = dbName
					result.Deleted = append(result.Deleted, "database:"+dbName)
				}
			}
		}
	}

	for _, configPath := range getDomainConfigCandidates(domain) {
		existed := fileExists(configPath)
		if err := removeIfExists(configPath); err != nil {
			return result, err
		}
		if existed {
			result.Deleted = append(result.Deleted, configPath)
		}
	}

	paths := getDomainPaths()
	logFiles := []string{
		filepath.Join(paths.nginxLogDir, domain+"_access.log"),
		filepath.Join(paths.nginxLogDir, domain+"_error.log"),
	}
	for _, logPath := range logFiles {
		existed := fileExists(logPath)
		if err := removeIfExists(logPath); err != nil {
			return result, err
		}
		if existed {
			result.Deleted = append(result.Deleted, logPath)
		}
	}

	if err := updateDomainNote(domain, ""); err != nil {
		return result, err
	}

	if runtime.GOOS != "windows" {
		if err := reloadNginx(); err != nil {
			return result, err
		}
		result.NginxReload = true
	}

	clearDomainCache()
	return result, nil
}

func getDomains(scan bool) []DomainInfo {
	notes := loadDomainNotes()
	stars := loadDomainStars()

	if !scan && cachedDomains != nil && time.Since(lastDomainCheck) < 30*time.Second {
		// Populate dynamic status for cached domains
		for i, d := range cachedDomains {
			cachedDomains[i].Note = notes[d.Domain]
			cachedDomains[i].IsStarred = stars[d.Domain]
		}
		return cachedDomains
	}

	paths := getDomainPaths()
	sitesEnabledDir := paths.sitesEnabledDir

	if runtime.GOOS == "windows" {
		_ = os.MkdirAll(sitesEnabledDir, 0755)
	}

	var list []DomainInfo
	files, err := os.ReadDir(sitesEnabledDir)
	if err != nil {
		return list
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if name == "default" || name == "phpmyadmin" {
			continue
		}
		domain := strings.TrimSuffix(name, ".conf")

		status := "online"
		code := 200

		if scan {
			// Lightweight HTTP scan
			client := http.Client{
				Timeout: 2 * time.Second,
			}
			urlStr := "http://" + domain
			resp, err := client.Get(urlStr)
			if err != nil {
				// Fallback to https
				urlStr = "https://" + domain
				resp, err = client.Get(urlStr)
			}

			if err != nil {
				status = "offline"
				code = 0
			} else {
				code = resp.StatusCode
				resp.Body.Close()
			}
		}

		list = append(list, DomainInfo{
			Domain:    domain,
			Status:    status,
			Code:      code,
			Note:      notes[domain],
			IsStarred: stars[domain],
		})
	}

	// Sort: Starred first, then alphabetically
	sort.Slice(list, func(i, j int) bool {
		if list[i].IsStarred && !list[j].IsStarred {
			return true
		}
		if !list[i].IsStarred && list[j].IsStarred {
			return false
		}
		return strings.ToLower(list[i].Domain) < strings.ToLower(list[j].Domain)
	})

	cachedDomains = list
	lastDomainCheck = time.Now()
	return list
}

func runCertbot(domain string) {
	if runtime.GOOS == "windows" {
		return
	}
	go func() {
		// Run certbot in background
		cmd := exec.Command("certbot", "--nginx", "-d", domain, "--non-interactive", "--agree-tos", "--register-unsafely-without-email")
		_ = cmd.Run()
		_ = reloadNginx()
	}()
}

func getSSLCertificates() []SSLCertInfo {
	var list []SSLCertInfo
	if runtime.GOOS == "windows" {
		now := time.Now()
		list = append(list, SSLCertInfo{
			Domain:     "testdomain.com",
			Issuer:     "Let's Encrypt",
			ExpiryDate: now.AddDate(0, 0, 45),
			DaysLeft:   45,
			IsExpired:  false,
		})
		list = append(list, SSLCertInfo{
			Domain:     "expired-domain.net",
			Issuer:     "ZeroSSL",
			ExpiryDate: now.AddDate(0, 0, -5),
			DaysLeft:   -5,
			IsExpired:  true,
		})
		return list
	}

	liveDir := "/etc/letsencrypt/live"
	entries, err := os.ReadDir(liveDir)
	if err != nil {
		return list
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		domain := entry.Name()
		if domain == "README" {
			continue
		}

		certPath := filepath.Join(liveDir, domain, "fullchain.pem")
		certData, err := os.ReadFile(certPath)
		if err != nil {
			continue
		}

		block, _ := pem.Decode(certData)
		if block == nil || block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}

		daysLeft := int(time.Until(cert.NotAfter).Hours() / 24)
		list = append(list, SSLCertInfo{
			Domain:     domain,
			Issuer:     cert.Issuer.CommonName,
			ExpiryDate: cert.NotAfter,
			DaysLeft:   daysLeft,
			IsExpired:  time.Now().After(cert.NotAfter),
		})
	}
	return list
}
