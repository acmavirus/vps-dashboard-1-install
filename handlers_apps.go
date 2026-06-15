package main

import (
	"encoding/json"
	"fmt"
	stdnet "net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

func registerAppsRoutes(api *gin.RouterGroup) {
	api.GET("/apps", func(c *gin.Context) {
		catalog := []StoreApp{
			{
				ID:          "nginx-proxy-manager",
				Name:        "Nginx Proxy Manager",
				Description: "Easy reverse proxy manager with automated SSL/TLS certificates via Let's Encrypt.",
				Category:    "Web Proxy",
				DefaultPort: "81",
				Image:       "jc21/nginx-proxy-manager:latest",
				Status:      "not_installed",
			},
			{
				ID:          "phpmyadmin",
				Name:        "phpMyAdmin",
				Description: "Web UI interface to manage MySQL and MariaDB databases easily.",
				Category:    "Database GUI",
				DefaultPort: "8080",
				Image:       "phpmyadmin:latest",
				Status:      "not_installed",
			},
			{
				ID:          "redis-cache",
				Name:        "Redis",
				Description: "High-performance in-memory key-value database and caching store.",
				Category:    "Database",
				DefaultPort: "6379",
				Image:       "redis:alpine",
				Status:      "not_installed",
			},
			{
				ID:          "postgres-db",
				Name:        "PostgreSQL",
				Description: "Robust open-source object-relational database management system.",
				Category:    "Database",
				DefaultPort: "5432",
				Image:       "postgres:alpine",
				Status:      "not_installed",
			},
			{
				ID:          "mongodb-db",
				Name:        "MongoDB",
				Description: "Popular document-oriented database for storing JSON-like documents.",
				Category:    "Database",
				DefaultPort: "27017",
				Image:       "mongo:latest",
				Status:      "not_installed",
			},
			{
				ID:          "wordpress-app",
				Name:        "WordPress",
				Description: "World's most popular blogging software and content management system.",
				Category:    "CMS",
				DefaultPort: "8081",
				Image:       "wordpress:latest",
				Status:      "not_installed",
			},
			{
				ID:          "joomla-app",
				Name:        "Joomla",
				Description: "A powerful, flexible, and feature-rich Content Management System (CMS).",
				Category:    "CMS",
				DefaultPort: "8082",
				Image:       "joomla:latest",
				Status:      "not_installed",
			},
			{
				ID:          "drupal-app",
				Name:        "Drupal",
				Description: "An open-source content management platform for high-performance websites.",
				Category:    "CMS",
				DefaultPort: "8083",
				Image:       "drupal:latest",
				Status:      "not_installed",
			},
			{
				ID:          "ghost-app",
				Name:        "Ghost",
				Description: "A professional headless Node.js blogging and publication platform.",
				Category:    "CMS",
				DefaultPort: "2368",
				Image:       "ghost:alpine",
				Status:      "not_installed",
			},
			{
				ID:          "prestashop-app",
				Name:        "PrestaShop",
				Description: "A popular, fully customizable open-source e-commerce solution.",
				Category:    "CMS",
				DefaultPort: "8084",
				Image:       "prestashop/prestashop:latest",
				Status:      "not_installed",
			},
		}

		cmd := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}|{{.State}}")
		output, err := cmd.CombinedOutput()
		statuses := make(map[string]string)
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				parts := strings.SplitN(line, "|", 2)
				if len(parts) == 2 {
					statuses[parts[0]] = parts[1]
				}
			}
		}

		metaList, _ := loadAppsMetadata()
		var finalApps []StoreApp
		installedIDs := make(map[string]bool)

		for _, m := range metaList {
			installedIDs[m.ID] = true

			if m.Domain != "" {
				continue
			}

			baseAppID := m.AppID
			if baseAppID == "" {
				baseAppID = m.ID
			}

			var baseApp StoreApp
			for _, ca := range catalog {
				if ca.ID == baseAppID {
					baseApp = ca
					break
				}
			}

			status := "stopped"
			if state, found := statuses[m.ID]; found && state == "running" {
				status = "running"
			}

			displayName := baseApp.Name
			if m.Domain != "" {
				displayName = fmt.Sprintf("%s (%s)", baseApp.Name, m.Domain)
			} else if m.ID != baseAppID {
				displayName = fmt.Sprintf("%s (%s)", baseApp.Name, m.ID)
			}

			finalApps = append(finalApps, StoreApp{
				ID:          m.ID,
				Name:        displayName,
				Description: baseApp.Description,
				Category:    baseApp.Category,
				DefaultPort: m.Port,
				Image:       baseApp.Image,
				Status:      status,
				Domain:      m.Domain,
			})
		}

		for _, ca := range catalog {
			if ca.ID == "nginx-proxy-manager" && installedIDs[ca.ID] {
				continue
			}
			ca.Status = "not_installed"
			finalApps = append(finalApps, ca)
		}

		c.JSON(200, finalApps)
	})

	api.POST("/apps/install", func(c *gin.Context) {
		var req struct {
			ID       string `json:"id"`
			Port     string `json:"port"`
			Password string `json:"password"`
			Domain   string `json:"domain"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		idPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
		if !idPattern.MatchString(req.ID) {
			c.JSON(400, gin.H{"error": "Invalid application ID"})
			return
		}

		portPattern := regexp.MustCompile(`^[0-9]+$`)
		if req.Port != "" && !portPattern.MatchString(req.Port) {
			c.JSON(400, gin.H{"error": "Invalid port number"})
			return
		}

		domainPattern := regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
		if req.Domain != "" && !domainPattern.MatchString(req.Domain) {
			c.JSON(400, gin.H{"error": "Invalid domain name"})
			return
		}

		var wpDBName, wpDBUser, wpDBPass string
		var runCmd *exec.Cmd
		port := req.Port
		if port == "" {
			switch req.ID {
			case "nginx-proxy-manager":
				port = "81"
			case "phpmyadmin":
				port = "8080"
			case "redis-cache":
				port = "6379"
			case "postgres-db":
				port = "5432"
			case "mongodb-db":
				port = "27017"
			case "wordpress-app":
				port = "8081"
			case "joomla-app":
				port = "8082"
			case "drupal-app":
				port = "8083"
			case "ghost-app":
				port = "2368"
			case "prestashop-app":
				port = "8084"
			}
		}

		containerID := req.ID
		if req.ID != "nginx-proxy-manager" {
			if req.Domain != "" {
				safeDomain := strings.ReplaceAll(req.Domain, ".", "-")
				containerID = fmt.Sprintf("%s-%s", req.ID, safeDomain)
			} else {
				containerID = fmt.Sprintf("%s-%s", req.ID, generateRandomString(6))
			}
		}

		metaList, _ := loadAppsMetadata()
		for _, m := range metaList {
			if m.ID == containerID {
				c.JSON(400, gin.H{"error": "An instance with this domain or name already exists."})
				return
			}
		}

		for _, m := range metaList {
			if m.Port == port {
				c.JSON(400, gin.H{"error": fmt.Sprintf("Cổng %s đã được sử dụng bởi ứng dụng khác (%s). Vui lòng chọn cổng khác.", port, m.ID)})
				return
			}
		}

		if port != "" {
			ln, err := stdnet.Listen("tcp", ":"+port)
			if err != nil {
				c.JSON(400, gin.H{"error": fmt.Sprintf("Cổng %s đang bị chiếm dụng trên hệ thống. Vui lòng chọn cổng khác.", port)})
				return
			}
			ln.Close()
		}

		_ = exec.Command("docker", "rm", "-f", containerID).Run()

		appRoot := filepath.Join("/home", req.Domain)
		if req.Domain == "" {
			appRoot = filepath.Join("/home", containerID)
		}
		if runtime.GOOS == "windows" {
			appRoot = filepath.Join(".", "logs", "www", containerID)
		}

		if req.ID == "wordpress-app" || req.ID == "joomla-app" || req.ID == "drupal-app" || req.ID == "ghost-app" || req.ID == "prestashop-app" {
			if err := os.MkdirAll(appRoot, 0755); err != nil {
				c.JSON(500, gin.H{"error": "Failed to create application root: " + err.Error()})
				return
			}
			_ = exec.Command("chown", "-R", "www-data:www-data", appRoot).Run()
		}

		switch req.ID {
		case "nginx-proxy-manager":
			if port == "" {
				port = "81"
			}
			runCmd = exec.Command("docker", "run", "-d",
				"--name", containerID,
				"--restart", "unless-stopped",
				"-p", "80:80",
				"-p", "443:443",
				"-p", port+":81",
				"-v", "/var/lib/nginx-proxy-manager:/data",
				"-v", "/var/lib/nginx-proxy-manager/letsencrypt:/etc/letsencrypt",
				"jc21/nginx-proxy-manager:latest")
		case "phpmyadmin":
			if port == "" {
				port = "8080"
			}
			runCmd = exec.Command("docker", "run", "-d",
				"--name", containerID,
				"--restart", "unless-stopped",
				"-p", port+":80",
				"-e", "PMA_ARBITRARY=1",
				"phpmyadmin:latest")
		case "redis-cache":
			if port == "" {
				port = "6379"
			}
			runCmd = exec.Command("docker", "run", "-d",
				"--name", containerID,
				"--restart", "unless-stopped",
				"-p", port+":6379",
				"-v", containerID+"_data:/data",
				"redis:alpine")
		case "postgres-db":
			if port == "" {
				port = "5432"
			}
			pwd := req.Password
			if pwd == "" {
				pwd = "postgres_secure_pass_2026"
			}
			runCmd = exec.Command("docker", "run", "-d",
				"--name", containerID,
				"--restart", "unless-stopped",
				"-p", port+":5432",
				"-v", containerID+"_data:/var/lib/postgresql/data",
				"-e", "POSTGRES_PASSWORD="+pwd,
				"postgres:alpine")
		case "mongodb-db":
			if port == "" {
				port = "27017"
			}
			pwd := req.Password
			if pwd == "" {
				pwd = "mongo_secure_pass_2026"
			}
			runCmd = exec.Command("docker", "run", "-d",
				"--name", containerID,
				"--restart", "unless-stopped",
				"-p", port+":27017",
				"-v", containerID+"_data:/data/db",
				"-e", "MONGO_INITDB_ROOT_USERNAME=admin",
				"-e", "MONGO_INITDB_ROOT_PASSWORD="+pwd,
				"mongo:latest")
		case "wordpress-app":
			if port == "" {
				port = "8081"
			}

			var dbErr error
			wpDBName, wpDBUser, wpDBPass, dbErr = provisionCMSDatabase("wp")
			if dbErr != nil {
				c.JSON(500, gin.H{"error": dbErr.Error()})
				return
			}

			dbConfig, _ := loadDBConfig()
			dbHostForDocker := dbConfig.Host
			if dbHostForDocker == "127.0.0.1" || dbHostForDocker == "localhost" {
				dbHostForDocker = "host.docker.internal"
			}
			dbHostPortForDocker := fmt.Sprintf("%s:%s", dbHostForDocker, dbConfig.Port)

			runCmd = exec.Command("docker", "run", "-d",
				"--name", containerID,
				"--add-host", "host.docker.internal:host-gateway",
				"--restart", "unless-stopped",
				"-p", port+":80",
				"-v", appRoot+":/var/www/html",
				"-e", "WORDPRESS_DB_HOST="+dbHostPortForDocker,
				"-e", "WORDPRESS_DB_USER="+wpDBUser,
				"-e", "WORDPRESS_DB_PASSWORD="+wpDBPass,
				"-e", "WORDPRESS_DB_NAME="+wpDBName,
				"wordpress:latest")
		case "joomla-app":
			if port == "" {
				port = "8082"
			}

			var dbErr error
			wpDBName, wpDBUser, wpDBPass, dbErr = provisionCMSDatabase("joomla")
			if dbErr != nil {
				c.JSON(500, gin.H{"error": dbErr.Error()})
				return
			}

			dbConfig, _ := loadDBConfig()
			dbHostForDocker := dbConfig.Host
			if dbHostForDocker == "127.0.0.1" || dbHostForDocker == "localhost" {
				dbHostForDocker = "host.docker.internal"
			}
			dbHostPortForDocker := fmt.Sprintf("%s:%s", dbHostForDocker, dbConfig.Port)

			runCmd = exec.Command("docker", "run", "-d",
				"--name", containerID,
				"--add-host", "host.docker.internal:host-gateway",
				"--restart", "unless-stopped",
				"-p", port+":80",
				"-v", appRoot+":/var/www/html",
				"-e", "JOOMLA_DB_HOST="+dbHostPortForDocker,
				"-e", "JOOMLA_DB_USER="+wpDBUser,
				"-e", "JOOMLA_DB_PASSWORD="+wpDBPass,
				"-e", "JOOMLA_DB_NAME="+wpDBName,
				"joomla:latest")
		case "drupal-app":
			if port == "" {
				port = "8083"
			}

			var dbErr error
			wpDBName, wpDBUser, wpDBPass, dbErr = provisionCMSDatabase("drupal")
			if dbErr != nil {
				c.JSON(500, gin.H{"error": dbErr.Error()})
				return
			}

			runCmd = exec.Command("docker", "run", "-d",
				"--name", containerID,
				"--add-host", "host.docker.internal:host-gateway",
				"--restart", "unless-stopped",
				"-p", port+":80",
				"-v", appRoot+":/var/www/html",
				"drupal:latest")
		case "ghost-app":
			if port == "" {
				port = "2368"
			}

			var dbErr error
			wpDBName, wpDBUser, wpDBPass, dbErr = provisionCMSDatabase("ghost")
			if dbErr != nil {
				c.JSON(500, gin.H{"error": dbErr.Error()})
				return
			}

			dbConfig, _ := loadDBConfig()
			dbHostForDocker := dbConfig.Host
			if dbHostForDocker == "127.0.0.1" || dbHostForDocker == "localhost" {
				dbHostForDocker = "host.docker.internal"
			}

			ghostURL := "http://localhost:" + port
			if req.Domain != "" {
				ghostURL = "http://" + req.Domain
			}

			runCmd = exec.Command("docker", "run", "-d",
				"--name", containerID,
				"--add-host", "host.docker.internal:host-gateway",
				"--restart", "unless-stopped",
				"-p", port+":2368",
				"-v", appRoot+":/var/lib/ghost/content",
				"-e", "url="+ghostURL,
				"-e", "database__client=mysql",
				"-e", "database__connection__host="+dbHostForDocker,
				"-e", "database__connection__port="+dbConfig.Port,
				"-e", "database__connection__user="+wpDBUser,
				"-e", "database__connection__password="+wpDBPass,
				"-e", "database__connection__database="+wpDBName,
				"ghost:alpine")
		case "prestashop-app":
			if port == "" {
				port = "8084"
			}

			var dbErr error
			wpDBName, wpDBUser, wpDBPass, dbErr = provisionCMSDatabase("presta")
			if dbErr != nil {
				c.JSON(500, gin.H{"error": dbErr.Error()})
				return
			}

			dbConfig, _ := loadDBConfig()
			dbHostForDocker := dbConfig.Host
			if dbHostForDocker == "127.0.0.1" || dbHostForDocker == "localhost" {
				dbHostForDocker = "host.docker.internal"
			}
			dbHostPortForDocker := fmt.Sprintf("%s:%s", dbHostForDocker, dbConfig.Port)

			os.MkdirAll(appRoot, 0755)
			runCmd = exec.Command("docker", "run", "-d",
				"--name", containerID,
				"--add-host", "host.docker.internal:host-gateway",
				"--restart", "unless-stopped",
				"-p", port+":80",
				"-v", appRoot+":/var/www/html",
				"-e", "DB_SERVER="+dbHostPortForDocker,
				"-e", "DB_USER="+wpDBUser,
				"-e", "DB_PASSWD="+wpDBPass,
				"-e", "DB_NAME="+wpDBName,
				"prestashop/prestashop:latest")
		default:
			c.JSON(400, gin.H{"error": "Unsupported application ID"})
			return
		}

		output, err := runCmd.CombinedOutput()
		if err != nil {
			if wpDBName != "" {
				_, _ = runSQLCommand(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%';", wpDBUser))
				_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", wpDBName))
				_, _ = runSQLCommand("FLUSH PRIVILEGES;")
			}
			c.JSON(500, gin.H{"error": fmt.Sprintf("%s: %s", err.Error(), strings.TrimSpace(string(output)))})
			return
		}

		updated := false
		newMeta := AppMetadata{
			ID:     containerID,
			AppID:  req.ID,
			Domain: req.Domain,
			Port:   port,
			DBName: wpDBName,
			DBUser: wpDBUser,
			DBPass: wpDBPass,
		}
		for i, m := range metaList {
			if m.ID == containerID {
				metaList[i] = newMeta
				updated = true
				break
			}
		}
		if !updated {
			metaList = append(metaList, newMeta)
		}
		_ = saveAppsMetadata(metaList)

		if req.Domain != "" {
			err = createNginxProxy(req.Domain, port)
			if err != nil {
				c.JSON(200, gin.H{
					"status":       "ok",
					"container_id": strings.TrimSpace(string(output)),
					"warning":      "Container started, but Nginx proxy creation failed: " + err.Error(),
				})
				return
			}
			runCertbot(req.Domain)

			if wpDBName != "" {
				note := fmt.Sprintf("CMS: %s | DB: %s | User: %s | Pass: %s", req.ID, wpDBName, wpDBUser, wpDBPass)
				_ = updateDomainNote(req.Domain, note)
			}
		}

		c.JSON(200, gin.H{"status": "ok", "container_id": strings.TrimSpace(string(output))})
	})

	api.POST("/apps/uninstall", func(c *gin.Context) {
		var req struct {
			ID string `json:"id"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		idPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
		if !idPattern.MatchString(req.ID) {
			c.JSON(400, gin.H{"error": "Invalid application ID"})
			return
		}

		cmd := exec.Command("docker", "rm", "-f", req.ID)
		output, err := cmd.CombinedOutput()
		if err != nil && !strings.Contains(string(output), "No such container") {
			c.JSON(500, gin.H{"error": fmt.Sprintf("%s: %s", err.Error(), strings.TrimSpace(string(output)))})
			return
		}

		_ = exec.Command("docker", "volume", "rm", req.ID+"_data").Run()

		metaList, _ := loadAppsMetadata()
		var remainingMeta []AppMetadata
		for _, m := range metaList {
			if m.ID == req.ID {
				if m.Domain != "" {
					_ = deleteNginxProxy(m.Domain)
				}
				if m.DBName != "" {
					_, _ = runSQLCommand(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%';", m.DBUser))
					_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", m.DBName))
					_, _ = runSQLCommand("FLUSH PRIVILEGES;")
				}
			} else {
				remainingMeta = append(remainingMeta, m)
			}
		}
		_ = saveAppsMetadata(remainingMeta)

		c.JSON(200, gin.H{"status": "ok"})
	})
}

func getAppsMetadataPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(".", "data", "apps-metadata.json")
	}
	return filepath.Join("/usr/local/bin", "data", "apps-metadata.json")
}

func loadAppsMetadata() ([]AppMetadata, error) {
	var list []AppMetadata
	path := getAppsMetadataPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return list, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &list)
	return list, err
}

func saveAppsMetadata(list []AppMetadata) error {
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getAppsMetadataPath(), data, 0600)
}

func createNginxProxy(domain string, port string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	paths := getDomainPaths()
	accLogPath := filepath.ToSlash(filepath.Join(paths.nginxLogDir, domain+"_access.log"))
	errLogPath := filepath.ToSlash(filepath.Join(paths.nginxLogDir, domain+"_error.log"))

	configContent := fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    access_log %s;
    error_log %s;

    location ~ /\.(env|git|ht|svn) {
        deny all;
    }

    location / {
        proxy_pass http://127.0.0.1:%s;
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
`, domain, accLogPath, errLogPath, port)

	availablePath := filepath.Join(paths.sitesAvailableDir, domain+".conf")
	enabledPath := filepath.Join(paths.sitesEnabledDir, domain+".conf")

	err := os.WriteFile(availablePath, []byte(configContent), 0644)
	if err != nil {
		return err
	}

	_ = os.Remove(enabledPath)
	err = os.Symlink(availablePath, enabledPath)
	if err != nil {
		return err
	}

	return reloadNginx()
}

func deleteNginxProxy(domain string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	paths := getDomainPaths()
	availablePath := filepath.Join(paths.sitesAvailableDir, domain+".conf")
	enabledPath := filepath.Join(paths.sitesEnabledDir, domain+".conf")

	_ = os.Remove(availablePath)
	_ = os.Remove(enabledPath)

	accLogPath := filepath.Join(paths.nginxLogDir, domain+"_access.log")
	errLogPath := filepath.Join(paths.nginxLogDir, domain+"_error.log")
	_ = os.Remove(accLogPath)
	_ = os.Remove(errLogPath)

	return reloadNginx()
}
