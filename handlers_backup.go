package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type BackupConfig struct {
	Provider     string `json:"provider"` // "local", "s3", "gdrive"
	S3AccessKey  string `json:"s3_access_key"`
	S3SecretKey  string `json:"s3_secret_key"`
	S3Bucket     string `json:"s3_bucket"`
	S3Endpoint   string `json:"s3_endpoint"`
	S3Region     string `json:"s3_region"`
	GDriveFolder string `json:"gdrive_folder"`
}

type BackupHistoryEntry struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`   // "site", "database"
	Target    string    `json:"target"` // domain or db name
	File      string    `json:"file"`
	Size      int64     `json:"size"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"` // "success", "failed"
	CloudSync string    `json:"cloud_sync"` // "synced", "pending", "none"
}

func registerBackupRoutes(api *gin.RouterGroup) {
	// 1. Get Backup Config
	api.GET("/backup/config", func(c *gin.Context) {
		cfg := loadBackupConfig()
		c.JSON(200, cfg)
	})

	// 2. Save Backup Config
	api.POST("/backup/config", func(c *gin.Context) {
		var cfg BackupConfig
		if err := c.BindJSON(&cfg); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		err := saveBackupConfig(cfg)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 3. List Backups
	api.GET("/backup/list", func(c *gin.Context) {
		backups := getBackupList()
		c.JSON(200, backups)
	})

	// 4. Trigger Backup Now
	api.POST("/backup/run", func(c *gin.Context) {
		var req struct {
			Type   string `json:"type"`   // "site", "database"
			Target string `json:"target"` // domain name or db name
		}
		if err := c.BindJSON(&req); err != nil || req.Type == "" || req.Target == "" {
			c.JSON(400, gin.H{"error": "Type and Target are required"})
			return
		}

		var file string
		var err error

		if req.Type == "site" {
			file, err = runSiteBackup(req.Target)
		} else if req.Type == "database" {
			file, err = runBackup(req.Target) // Using existing runBackup function in handlers_database.go
		} else {
			c.JSON(400, gin.H{"error": "Invalid backup type"})
			return
		}

		if err != nil {
			c.JSON(500, gin.H{"error": "Backup failed: " + err.Error()})
			return
		}

		// Save backup details to history
		cfg := loadBackupConfig()
		status := "success"
		cloudSync := "none"

		size := int64(0)
		if info, err := os.Stat(file); err == nil {
			size = info.Size()
		}

		if cfg.Provider == "s3" || cfg.Provider == "gdrive" {
			cloudSync = "pending"
			// Asynchronously sync to cloud
			go func(filePath string, provider string) {
				time.Sleep(2 * time.Second) // Let disk settle
				err := syncBackupToCloud(filePath, provider)
				if err != nil {
					logSecurityEventSQL(SecurityLogEntry{
						IP:        "-",
						Timestamp: time.Now(),
						URI:       "Backup Cloud Sync",
						Domain:    req.Target,
						UserAgent: "-",
						Action:    "Cloud Sync Failed: " + err.Error(),
					})
				} else {
					logSecurityEventSQL(SecurityLogEntry{
						IP:        "-",
						Timestamp: time.Now(),
						URI:       "Backup Cloud Sync",
						Domain:    req.Target,
						UserAgent: "-",
						Action:    "Cloud Sync Success",
					})
				}
			}(file, cfg.Provider)
		}

		logSecurityEventSQL(SecurityLogEntry{
			IP:        "-",
			Timestamp: time.Now(),
			URI:       "Backup Manual",
			Domain:    req.Target,
			UserAgent: "-",
			Action:    fmt.Sprintf("Backup %s success. File: %s", req.Type, filepath.Base(file)),
		})

		c.JSON(200, gin.H{
			"status":     status,
			"file":       filepath.Base(file),
			"size":       size,
			"cloud_sync": cloudSync,
		})
	})
}

func loadBackupConfig() BackupConfig {
	var cfg BackupConfig
	val := getSetting("backup_config", "")
	if val == "" {
		cfg.Provider = "local"
		return cfg
	}
	_ = json.Unmarshal([]byte(val), &cfg)
	return cfg
}

func saveBackupConfig(cfg BackupConfig) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return saveSetting("backup_config", string(data))
}

func runSiteBackup(domain string) (string, error) {
	// Find web root path
	configPath, err := findDomainConfigPath(domain)
	if err != nil {
		return "", err
	}

	rootPath, err := parseNginxRoot(configPath)
	if err != nil {
		return "", fmt.Errorf("cannot parse root from nginx config: %w", err)
	}

	appRoot := getAppRoot(rootPath)

	backupDir := "/var/www/backups"
	if runtime.GOOS == "windows" {
		backupDir = filepath.Join(".", "logs", "backups")
	}
	_ = os.MkdirAll(backupDir, 0755)

	backupFile := filepath.Join(backupDir, fmt.Sprintf("site_%s_%s.zip", domain, time.Now().Format("20060102_150405")))

	// Zip directory
	err = zipDirectory(appRoot, backupFile)
	if err != nil {
		return "", err
	}

	return backupFile, nil
}

func syncBackupToCloud(filePath string, provider string) error {
	if runtime.GOOS == "windows" {
		log.Printf("[Backup SIMULATION] Uploaded %s to %s\n", filePath, provider)
		return nil
	}

	// For Linux VPS, we trigger rclone or a shell backup script if configured
	// Standard rclone config is managed under /root/.config/rclone/rclone.conf
	// We run: rclone copy <filePath> acmadash_remote:<bucket_or_folder>
	remoteName := "acmadash_backup"
	cfg := loadBackupConfig()
	
	var rcloneArgs []string
	if provider == "s3" {
		// Try to run rclone if configured
		rcloneArgs = []string{"copy", filePath, fmt.Sprintf("%s:%s", remoteName, cfg.S3Bucket)}
	} else if provider == "gdrive" {
		rcloneArgs = []string{"copy", filePath, fmt.Sprintf("%s:%s", remoteName, cfg.GDriveFolder)}
	} else {
		return nil
	}

	cmd := exec.Command("rclone", rcloneArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rclone upload failed: %s %w", string(output), err)
	}

	return nil
}

func getBackupList() []BackupHistoryEntry {
	var list []BackupHistoryEntry
	backupDir := "/var/www/backups"
	if runtime.GOOS == "windows" {
		backupDir = filepath.Join(".", "logs", "backups")
	}

	files, err := os.ReadDir(backupDir)
	if err != nil {
		return list
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		info, err := f.Info()
		if err != nil {
			continue
		}

		name := f.Name()
		bType := "unknown"
		target := "unknown"

		if strings.HasPrefix(name, "site_") {
			bType = "site"
			parts := strings.Split(name, "_")
			if len(parts) >= 2 {
				target = parts[1]
			}
		} else {
			bType = "database"
			parts := strings.Split(name, "_")
			if len(parts) >= 1 {
				target = parts[0]
			}
		}

		list = append(list, BackupHistoryEntry{
			ID:        name,
			Type:      bType,
			Target:    target,
			File:      name,
			Size:      info.Size(),
			Timestamp: info.ModTime(),
			Status:    "success",
			CloudSync: "none",
		})
	}

	return list
}
