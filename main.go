package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/v4/cpu"
)

var Version = "v3.1.0"

//go:embed all:frontend/dist
var frontendFS embed.FS

var (
	adminUser = "admin"
	adminPass = "h5jH7Gv|5m+0"
	authToken = "acmadash_secret_token_2024"

	lastCpuAlert  time.Time
	lastDdosAlert time.Time

	cachedDomains   []DomainInfo
	lastDomainCheck time.Time
	cachedCPUModel  string
	cachedCPUCores  int

	cachedSoftware    interface{}
	lastSoftwareCheck time.Time
)

var (
	domainNamePattern = regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
	dbNamePattern     = regexp.MustCompile(`^[a-zA-Z0-9_.$-]+$`)
)

func autoHealNginxLogs() {
	paths := getDomainPaths()
	nginxDir := paths.nginxLogDir
	sitesAvailableDir := paths.sitesAvailableDir
	sitesEnabledDir := paths.sitesEnabledDir

	if runtime.GOOS == "windows" {
		_ = os.MkdirAll(nginxDir, 0755)
		_ = os.MkdirAll(sitesAvailableDir, 0755)
		_ = os.MkdirAll(sitesEnabledDir, 0755)
	}

	files, err := os.ReadDir(sitesAvailableDir)
	if err != nil {
		return
	}

	nginxChanged := false

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if name == "default" || name == "phpmyadmin" {
			continue
		}

		domain := strings.TrimSuffix(name, ".conf")
		configPath := filepath.Join(sitesAvailableDir, name)

		contentBytes, err := os.ReadFile(configPath)
		if err != nil {
			continue
		}
		content := string(contentBytes)

		// Check if it's an active config (has server { block)
		if !strings.Contains(strings.ToLower(content), "server {") && !strings.Contains(strings.ToLower(content), "server{") {
			continue
		}

		// Check if log files are configured
		hasAccess := strings.Contains(content, "access_log")
		hasError := strings.Contains(content, "error_log")

		// If either is missing, we insert it into the config file
		if !hasAccess || !hasError {
			idx := strings.Index(strings.ToLower(content), "server {")
			insertLen := len("server {")
			if idx == -1 {
				idx = strings.Index(strings.ToLower(content), "server{")
				insertLen = len("server{")
			}

			if idx != -1 {
				var insertLines string
				if !hasAccess {
					accLogPath := filepath.Join(nginxDir, domain+"_access.log")
					accLogPathStr := filepath.ToSlash(accLogPath)
					insertLines += fmt.Sprintf("\n    access_log %s;", accLogPathStr)
				}
				if !hasError {
					errLogPath := filepath.Join(nginxDir, domain+"_error.log")
					errLogPathStr := filepath.ToSlash(errLogPath)
					insertLines += fmt.Sprintf("\n    error_log %s;", errLogPathStr)
				}

				newContent := content[:idx+insertLen] + insertLines + content[idx+insertLen:]
				if err := os.WriteFile(configPath, []byte(newContent), 0644); err == nil {
					nginxChanged = true
				}
			}
		}

		accLogPath := filepath.Join(nginxDir, domain+"_access.log")
		errLogPath := filepath.Join(nginxDir, domain+"_error.log")
		ensureLogFileExists(accLogPath)
		ensureLogFileExists(errLogPath)
	}

	if nginxChanged && runtime.GOOS != "windows" {
		if err := exec.Command("nginx", "-t").Run(); err == nil {
			_ = exec.Command("systemctl", "reload", "nginx").Run()
		}
	}
}

func main() {
	_ = godotenv.Load(".env")

	// Initialize SQLite Database
	initDB()

	// Scan and auto-heal missing Nginx logs
	autoHealNginxLogs()

	// Start Intrusion Prevention System (IPS) background routine
	go startIntrusionPreventionSystem()

	// Start background worker to record historical metrics (every 5 minutes)
	go startHistoricalMetricsCollector()

	// Cache CPU hardware specs at startup
	if info, err := cpu.Info(); err == nil && len(info) > 0 {
		cachedCPUModel = info[0].ModelName
	} else {
		cachedCPUModel = "Unknown CPU"
	}
	cachedCPUCores, _ = cpu.Counts(true)
	if cachedCPUCores <= 0 {
		cachedCPUCores = 1
	}

	vFlag := flag.Bool("v", false, "Version")
	flag.Parse()
	if *vFlag {
		fmt.Printf("Version: %s\n", Version)
		return
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// --- Authentication Configuration ---
	if u := os.Getenv("ADMIN_USER"); u != "" {
		adminUser = u
	}
	if p := os.Getenv("ADMIN_PASS"); p != "" {
		adminPass = p
	}
	
	adminUser = getSetting("admin_user", adminUser)
	adminPass = getSetting("admin_pass", adminPass)
	
	_ = saveSetting("admin_user", adminUser)
	_ = saveSetting("admin_pass", adminPass)

	if t := os.Getenv("AUTH_TOKEN"); t != "" {
		authToken = t
	}

	// Register Auth routes (unprotected endpoints)
	registerAuthRoutes(r)

	// API - Protected Group
	api := r.Group("/api")
	api.Use(authMiddleware)
	{
		registerSystemRoutes(api)
		registerDockerRoutes(api)
		registerDomainRoutes(api)
		registerDatabaseRoutes(api)
		registerSecurityRoutes(api)
		registerFilesRoutes(api)
		registerAppsRoutes(api)
		registerCronRoutes(api)
		registerTerminalRoutes(api)
		registerPHPRoutes(api)
		registerBackupRoutes(api)
		registerProtectedAuthRoutes(api)
	}

	// Static Files Fallback
	publicFS, _ := fs.Sub(frontendFS, "frontend/dist")
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			c.JSON(404, gin.H{"error": "Not Found"})
			return
		}
		trimPath := strings.TrimPrefix(path, "/")
		if trimPath == "" || trimPath == "/" {
			trimPath = "index.html"
		}
		data, err := fs.ReadFile(publicFS, trimPath)
		if err != nil {
			data, _ = fs.ReadFile(publicFS, "index.html")
			trimPath = "index.html"
		}
		contentType := "text/plain"
		switch {
		case strings.HasSuffix(trimPath, ".html"):
			contentType = "text/html"
		case strings.HasSuffix(trimPath, ".js"):
			contentType = "application/javascript"
		case strings.HasSuffix(trimPath, ".css"):
			contentType = "text/css"
		case strings.HasSuffix(trimPath, ".svg"):
			contentType = "image/svg+xml"
		}
		c.Data(200, contentType, data)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8900"
	}
	log.Printf("🚀 AcmaDash %s running on :%s\n", Version, port)
	r.Run(":" + port)
}
