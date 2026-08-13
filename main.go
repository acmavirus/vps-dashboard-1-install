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

func ensureDefaultNginxPage() {
	paths := getDomainPaths()
	sitesAvailableDir := paths.sitesAvailableDir
	sitesEnabledDir := paths.sitesEnabledDir
	defaultHtmlDir := "/var/www/default"

	if runtime.GOOS == "windows" {
		defaultHtmlDir = "./logs/www-default"
		_ = os.MkdirAll(sitesAvailableDir, 0755)
		_ = os.MkdirAll(sitesEnabledDir, 0755)
	}

	_ = os.MkdirAll(defaultHtmlDir, 0755)

	htmlPath := filepath.Join(defaultHtmlDir, "index.html")
	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		defaultHtmlContent := `<!DOCTYPE html>
<html lang="vi">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Tên miền chưa cấu hình</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background-color: #0f172a;
            color: #f8fafc;
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            padding: 20px;
        }
        .card {
            background-color: #1e293b;
            border: 1px solid #334155;
            border-radius: 12px;
            padding: 40px;
            max-width: 480px;
            width: 100%;
            text-align: center;
            box-shadow: 0 10px 25px rgba(0,0,0,0.3);
        }
        .icon {
            font-size: 48px;
            margin-bottom: 16px;
        }
        h1 {
            font-size: 20px;
            font-weight: 600;
            color: #38bdf8;
            margin-bottom: 12px;
        }
        p {
            color: #94a3b8;
            font-size: 14px;
            line-height: 1.6;
        }
    </style>
</head>
<body>
    <div class="card">
        <div class="icon">🌐</div>
        <h1>Tên Miền Chưa Cấu Hình</h1>
        <p>Tên miền này đã trỏ thành công về IP Server, nhưng chưa được thiết lập Virtual Host trên Nginx.</p>
    </div>
</body>
</html>`
		_ = os.WriteFile(htmlPath, []byte(defaultHtmlContent), 0644)
	}

	configPath := filepath.Join(sitesAvailableDir, "00-default.conf")
	configCreated := false
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		rootPath := filepath.ToSlash(defaultHtmlDir)
		var configContent string
		if runtime.GOOS == "windows" {
			configContent = fmt.Sprintf(`server {
    listen 80 default_server;

    server_name _;

    root %s;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }
}
`, rootPath)
		} else {
			configContent = fmt.Sprintf(`server {
    listen 80 default_server;
    listen [::]:80 default_server;

    server_name _;

    root %s;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }

    access_log /var/log/nginx/default_access.log;
    error_log /var/log/nginx/default_error.log;
}
`, rootPath)
		}
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err == nil {
			configCreated = true
		}
	}

	legacyDefaultLink := filepath.Join(sitesEnabledDir, "default")
	if _, err := os.Lstat(legacyDefaultLink); err == nil {
		_ = os.Remove(legacyDefaultLink)
		configCreated = true
	}

	enabledLink := filepath.Join(sitesEnabledDir, "00-default.conf")
	if _, err := os.Lstat(enabledLink); os.IsNotExist(err) {
		if runtime.GOOS != "windows" {
			if err := os.Symlink(configPath, enabledLink); err == nil {
				configCreated = true
			}
		} else {
			configBytes, _ := os.ReadFile(configPath)
			_ = os.WriteFile(enabledLink, configBytes, 0644)
			configCreated = true
		}
	}

	if configCreated && runtime.GOOS != "windows" {
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

	// Ensure Nginx default catch-all page exists
	ensureDefaultNginxPage()

	// Start Intrusion Prevention System (IPS) background routine
	go startIntrusionPreventionSystem()

	// Start background worker to record historical metrics (every 5 minutes)
	go startHistoricalMetricsCollector()

	// Start interactive Telegram Bot
	go startTelegramBot()

	// Start background CPU Usage Monitor (1s sampling)
	go startCpuMonitor()

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
		registerFtpRoutes(api)
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
