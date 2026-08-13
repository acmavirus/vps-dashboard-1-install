package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

type FtpUserRecord struct {
	Username string `json:"username"`
	Path     string `json:"path"`
	Status   string `json:"status"`
}

type AddFtpRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Path     string `json:"path" binding:"required"`
}

type FtpUserActionRequest struct {
	Username string `json:"username" binding:"required"`
}

type ChangeFtpPasswordRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func registerFtpRoutes(api *gin.RouterGroup) {
	ftp := api.Group("/ftp")
	{
		ftp.GET("", getFtpUsersHandler)
		ftp.POST("/add", addFtpUserHandler)
		ftp.POST("/delete", deleteFtpUserHandler)
		ftp.POST("/toggle", toggleFtpUserHandler)
		ftp.POST("/password", changeFtpPasswordHandler)
	}
}

// getFtpUsersHandler lists all FTP users from pure-pw & DB
func getFtpUsersHandler(c *gin.Context) {
	if runtime.GOOS == "windows" {
		users, err := getFtpUsersFromDB()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, users)
		return
	}

	// On Linux: Query pure-pw list
	cmd := exec.Command("pure-pw", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to DB if pure-pw fails
		users, _ := getFtpUsersFromDB()
		c.JSON(http.StatusOK, users)
		return
	}

	lines := strings.Split(string(output), "\n")
	var result []FtpUserRecord
	dbUsers, _ := getFtpUsersFromDBMap()

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			username := parts[0]
			path := parts[1]
			// Clean pureftpd path suffix like /./ or /.
			path = strings.TrimSuffix(path, "/./")
			path = strings.TrimSuffix(path, "/.")
			if path == "" {
				path = "/"
			}

			status := "active"
			if dbStatus, exists := dbUsers[username]; exists {
				status = dbStatus
			} else {
				// Record in DB if missing
				_ = saveFtpUserToDB(username, path, status)
			}

			result = append(result, FtpUserRecord{
				Username: username,
				Path:     path,
				Status:   status,
			})
		}
	}

	c.JSON(http.StatusOK, result)
}

// addFtpUserHandler creates a new virtual FTP user
func addFtpUserHandler(c *gin.Context) {
	var req AddFtpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên người dùng, mật khẩu và đường dẫn không được để trống"})
		return
	}

	username := strings.TrimSpace(req.Username)
	password := req.Password
	path := strings.TrimSpace(req.Path)

	if username == "" || len(password) < 6 || path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu nhập không hợp lệ (mật khẩu tối thiểu 6 ký tự)"})
		return
	}

	if runtime.GOOS != "windows" {
		// Run pure-pw useradd username -u 33 -g 33 -d path -m
		// Pass password twice to stdin
		cmdStr := fmt.Sprintf("printf '%%s\\n%%s\\n' %s %s | pure-pw useradd %s -u 33 -g 33 -d %s -m",
			shellEscape(password), shellEscape(password), shellEscape(username), shellEscape(path))

		cmd := exec.Command("bash", "-c", cmdStr)
		out, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Lỗi tạo tài khoản FTP: %s (%s)", err.Error(), string(out))})
			return
		}
	}

	// Save to DB
	if err := saveFtpUserToDB(username, path, "active"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu thông tin tài khoản vào cơ sở dữ liệu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã tạo tài khoản FTP thành công"})
}

// deleteFtpUserHandler removes an FTP user
func deleteFtpUserHandler(c *gin.Context) {
	var req FtpUserActionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên tài khoản không hợp lệ"})
		return
	}

	username := strings.TrimSpace(req.Username)

	if runtime.GOOS != "windows" {
		cmdStr := fmt.Sprintf("pure-pw userdel %s -m", shellEscape(username))
		cmd := exec.Command("bash", "-c", cmdStr)
		out, err := cmd.CombinedOutput()
		if err != nil {
			// Log error but proceed to remove from DB if not found
			_ = out
		}
	}

	if err := deleteFtpUserFromDB(username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa tài khoản khỏi cơ sở dữ liệu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã xóa tài khoản FTP thành công"})
}

// toggleFtpUserHandler enables or disables an FTP user
func toggleFtpUserHandler(c *gin.Context) {
	var req FtpUserActionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên tài khoản không hợp lệ"})
		return
	}

	username := strings.TrimSpace(req.Username)
	dbMap, err := getFtpUsersFromDBMap()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc cơ sở dữ liệu"})
		return
	}

	currentStatus, exists := dbMap[username]
	if !exists {
		currentStatus = "active"
	}

	newStatus := "disabled"
	if currentStatus == "disabled" {
		newStatus = "active"
	}

	if runtime.GOOS != "windows" {
		timeRule := "0000-0000"
		if newStatus == "disabled" {
			timeRule = "0000-0001" // restrict access time to 1 minute a day (effectively disables user)
		}
		cmdStr := fmt.Sprintf("pure-pw usermod %s -z %s -m", shellEscape(username), timeRule)
		_ = exec.Command("bash", "-c", cmdStr).Run()
	}

	if err := updateFtpUserStatusDB(username, newStatus); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trạng thái"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "status": newStatus})
}

// changeFtpPasswordHandler updates password for an FTP user
func changeFtpPasswordHandler(c *gin.Context) {
	var req ChangeFtpPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên tài khoản hoặc mật khẩu (tối thiểu 6 ký tự) không hợp lệ"})
		return
	}

	username := strings.TrimSpace(req.Username)
	password := req.Password

	if runtime.GOOS != "windows" {
		cmdStr := fmt.Sprintf("printf '%%s\\n%%s\\n' %s %s | pure-pw passwd %s -m",
			shellEscape(password), shellEscape(password), shellEscape(username))
		cmd := exec.Command("bash", "-c", cmdStr)
		out, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Lỗi đổi mật khẩu FTP: %s", string(out))})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã đổi mật khẩu FTP thành công"})
}

// --- Helper shell escape ---
func shellEscape(arg string) string {
	return "'" + strings.ReplaceAll(arg, "'", "'\"'\"'") + "'"
}

// --- DB helpers for FTP ---
func getFtpUsersFromDB() ([]FtpUserRecord, error) {
	rows, err := DB.Query("SELECT username, path, status FROM ftp_users ORDER BY username ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []FtpUserRecord
	for rows.Next() {
		var r FtpUserRecord
		var path, status sql.NullString
		if err := rows.Scan(&r.Username, &path, &status); err != nil {
			return nil, err
		}
		r.Path = path.String
		r.Status = status.String
		if r.Status == "" {
			r.Status = "active"
		}
		list = append(list, r)
	}
	return list, nil
}

func getFtpUsersFromDBMap() (map[string]string, error) {
	rows, err := DB.Query("SELECT username, status FROM ftp_users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]string)
	for rows.Next() {
		var u, s sql.NullString
		if err := rows.Scan(&u, &s); err == nil && u.Valid {
			status := s.String
			if status == "" {
				status = "active"
			}
			res[u.String] = status
		}
	}
	return res, nil
}

func saveFtpUserToDB(username, path, status string) error {
	_, err := DB.Exec(`INSERT INTO ftp_users (username, path, status) VALUES (?, ?, ?)
		ON CONFLICT(username) DO UPDATE SET path = excluded.path, status = excluded.status`,
		username, path, status)
	return err
}

func updateFtpUserStatusDB(username, status string) error {
	_, err := DB.Exec("UPDATE ftp_users SET status = ? WHERE username = ?", status, username)
	return err
}

func deleteFtpUserFromDB(username string) error {
	_, err := DB.Exec("DELETE FROM ftp_users WHERE username = ?", username)
	return err
}
