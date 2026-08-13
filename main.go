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
	sslDir := "/etc/nginx/ssl"

	if runtime.GOOS == "windows" {
		defaultHtmlDir = "./logs/www-default"
		sslDir = "./logs/ssl"
		_ = os.MkdirAll(sitesAvailableDir, 0755)
		_ = os.MkdirAll(sitesEnabledDir, 0755)
	}

	_ = os.MkdirAll(defaultHtmlDir, 0755)
	_ = os.MkdirAll(sslDir, 0755)

	htmlPath := filepath.Join(defaultHtmlDir, "index.html")
	defaultHtmlContent := `<!DOCTYPE html>
<html lang="vi">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hệ Thống Đang Bảo Trì & Sắp Ra Mắt</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background-color: #0f172a;
            color: #f8fafc;
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            padding: 24px;
            overflow: hidden;
            position: relative;
        }
        .bg-glow {
            position: absolute;
            width: 400px;
            height: 400px;
            background: radial-gradient(circle, rgba(56, 189, 248, 0.15) 0%, rgba(99, 102, 241, 0.05) 50%, transparent 70%);
            border-radius: 50%;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            animation: pulse 6s ease-in-out infinite alternate;
            pointer-events: none;
        }
        @keyframes pulse {
            0% { transform: translate(-50%, -50%) scale(0.8); opacity: 0.5; }
            100% { transform: translate(-50%, -50%) scale(1.2); opacity: 1; }
        }
        .card {
            position: relative;
            background: rgba(30, 41, 59, 0.7);
            backdrop-filter: blur(16px);
            -webkit-backdrop-filter: blur(16px);
            border: 1px solid rgba(255, 255, 255, 0.1);
            border-radius: 20px;
            padding: 48px 36px;
            max-width: 520px;
            width: 100%;
            text-align: center;
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4);
            z-index: 10;
        }
        .badge {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            background: rgba(56, 189, 248, 0.1);
            border: 1px solid rgba(56, 189, 248, 0.3);
            color: #38bdf8;
            font-size: 13px;
            font-weight: 600;
            padding: 6px 14px;
            border-radius: 9999px;
            margin-bottom: 24px;
            letter-spacing: 0.5px;
            text-transform: uppercase;
        }
        .badge-dot {
            width: 8px;
            height: 8px;
            background-color: #38bdf8;
            border-radius: 50%;
            animation: blink 1.5s infinite;
        }
        @keyframes blink {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.3; }
        }
        .icon {
            font-size: 56px;
            margin-bottom: 20px;
            display: inline-block;
            animation: float 3s ease-in-out infinite;
        }
        @keyframes float {
            0%, 100% { transform: translateY(0); }
            50% { transform: translateY(-8px); }
        }
        h1 {
            font-size: 26px;
            font-weight: 700;
            background: linear-gradient(135deg, #ffffff 0%, #cbd5e1 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 14px;
            line-height: 1.3;
        }
        p {
            color: #94a3b8;
            font-size: 15px;
            line-height: 1.7;
            margin-bottom: 28px;
        }
        .footer-text {
            font-size: 13px;
            color: #64748b;
            border-top: 1px solid rgba(255, 255, 255, 0.06);
            padding-top: 20px;
        }
    </style>
</head>
<body>
    <div class="bg-glow"></div>
    <div class="card">
        <div class="badge">
            <span class="badge-dot"></span>
            Hệ Thống Đang Nâng Cấp
        </div>
        <div class="icon">🚀</div>
        <h1>Website Đang Bảo Trì & Sắp Ra Mắt</h1>
        <p>Hệ thống hiện đang được bảo trì định kỳ và nâng cấp tính năng mới. Chúng tôi sẽ trở lại trong thời gian sớm nhất!</p>
        <div class="footer-text">
            Cảm ơn sự kiên nhẫn của bạn. Vui lòng quay lại sau ít phút.
        </div>
    </div>
</body>
</html>`
	_ = os.WriteFile(htmlPath, []byte(defaultHtmlContent), 0644)

	certPath := filepath.Join(sslDir, "default.crt")
	keyPath := filepath.Join(sslDir, "default.key")

	if runtime.GOOS != "windows" {
		if _, err := os.Stat(certPath); os.IsNotExist(err) {
			_ = exec.Command("openssl", "req", "-x509", "-nodes", "-days", "3650", "-newkey", "rsa:2048",
				"-keyout", keyPath, "-out", certPath, "-subj", "/CN=default").Run()
		}
	} else {
		if _, err := os.Stat(certPath); os.IsNotExist(err) {
			_ = os.WriteFile(certPath, []byte("--- DUMMY CERT ---"), 0644)
			_ = os.WriteFile(keyPath, []byte("--- DUMMY KEY ---"), 0644)
		}
	}

	configPath := filepath.Join(sitesAvailableDir, "00-default.conf")
	configCreated := false

	existingContent, _ := os.ReadFile(configPath)
	contentStr := string(existingContent)

	if contentStr == "" || !strings.Contains(contentStr, "listen 443") || !strings.Contains(contentStr, "charset utf-8") {
		rootPath := filepath.ToSlash(defaultHtmlDir)
		certPathStr := filepath.ToSlash(certPath)
		keyPathStr := filepath.ToSlash(keyPath)
		var configContent string
		if runtime.GOOS == "windows" {
			configContent = fmt.Sprintf(`server {
    listen 80 default_server;

    server_name _;
    charset utf-8;

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
    charset utf-8;

    root %s;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }

    access_log /var/log/nginx/default_access.log;
    error_log /var/log/nginx/default_error.log;
}

server {
    listen 443 ssl default_server;
    listen [::]:443 ssl default_server;

    server_name _;
    charset utf-8;

    ssl_certificate %s;
    ssl_certificate_key %s;

    root %s;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }

    access_log /var/log/nginx/default_access.log;
    error_log /var/log/nginx/default_error.log;
}
`, rootPath, certPathStr, keyPathStr, rootPath)
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
