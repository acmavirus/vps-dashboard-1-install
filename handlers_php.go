package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type PHPVersionInfo struct {
	Version     string `json:"version"`
	Status      string `json:"status"` // running, stopped, not_installed
	IsContainer bool   `json:"is_container"`
	ContainerID string `json:"container_id"`
}

type ContainerPHPInfo struct {
	IsPHP   bool
	Version string
}

var (
	phpContainerCache      = make(map[string]ContainerPHPInfo)
	phpContainerCacheMutex sync.RWMutex
)

func getPHPCache(name string) (ContainerPHPInfo, bool) {
	phpContainerCacheMutex.RLock()
	defer phpContainerCacheMutex.RUnlock()
	info, exists := phpContainerCache[name]
	return info, exists
}

func setPHPCache(name string, info ContainerPHPInfo) {
	phpContainerCacheMutex.Lock()
	defer phpContainerCacheMutex.Unlock()
	phpContainerCache[name] = info
}

func isLikelyPHPImage(image string) bool {
	img := strings.ToLower(image)
	return strings.Contains(img, "php") ||
		strings.Contains(img, "wordpress") ||
		strings.Contains(img, "joomla") ||
		strings.Contains(img, "drupal") ||
		strings.Contains(img, "prestashop")
}

func detectContainerPHP(name string) ContainerPHPInfo {
	if info, exists := getPHPCache(name); exists {
		return info
	}

	cmd := exec.Command("docker", "exec", name, "php", "-r", "echo PHP_VERSION;")
	out, err := cmd.Output()
	if err != nil {
		// Do not cache false permanently if container is simply not running or not found
		return ContainerPHPInfo{IsPHP: false, Version: ""}
	}

	ver := strings.TrimSpace(string(out))
	if ver == "" {
		info := ContainerPHPInfo{IsPHP: false, Version: ""}
		setPHPCache(name, info)
		return info
	}

	info := ContainerPHPInfo{IsPHP: true, Version: ver}
	setPHPCache(name, info)
	return info
}

func getDockerPHPVersions() []PHPVersionInfo {
	if runtime.GOOS == "windows" {
		return []PHPVersionInfo{
			{
				Version:     "wordpress-app (PHP 8.2)",
				Status:      "running",
				IsContainer: true,
				ContainerID: "wordpress-app",
			},
			{
				Version:     "prestashop-app (PHP 8.1)",
				Status:      "stopped",
				IsContainer: true,
				ContainerID: "prestashop-app",
			},
		}
	}

	var list []PHPVersionInfo

	cmd := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}|{{.State}}|{{.Image}}")
	out, err := cmd.Output()
	if err != nil {
		return list
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		name := parts[0]
		state := parts[1]
		image := parts[2]

		isRunning := state == "running"

		var phpInfo ContainerPHPInfo
		if isRunning {
			phpInfo = detectContainerPHP(name)
		} else {
			if cachedInfo, exists := getPHPCache(name); exists {
				phpInfo = cachedInfo
			} else if isLikelyPHPImage(image) {
				phpInfo = ContainerPHPInfo{IsPHP: true, Version: "Unknown"}
			}
		}

		if phpInfo.IsPHP {
			statusStr := "stopped"
			if isRunning {
				statusStr = "running"
			}
			displayName := name
			if phpInfo.Version != "" && phpInfo.Version != "Unknown" {
				displayName = fmt.Sprintf("%s (PHP %s)", name, phpInfo.Version)
			}
			list = append(list, PHPVersionInfo{
				Version:     displayName,
				Status:      statusStr,
				IsContainer: true,
				ContainerID: name,
			})
		}
	}

	return list
}

func findContainerIniScanDir(container string) string {
	cmd := exec.Command("docker", "exec", container, "php", "--ini")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Scan for additional .ini files in") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				path := strings.TrimSpace(parts[1])
				if path != "(none)" && path != "" {
					return path
				}
			}
		}
	}
	return ""
}

func findContainerLoadedIni(container string) string {
	cmd := exec.Command("docker", "exec", container, "php", "-r", "echo php_ini_loaded_file();")
	out, err := cmd.Output()
	if err == nil {
		path := strings.TrimSpace(string(out))
		if path != "" && path != "false" {
			return path
		}
	}
	return ""
}

func getContainerPHPExtensions(container string) []PHPExtension {
	standardExts := []string{"opcache", "redis", "imagick", "gd", "pdo_mysql", "curl", "mbstring", "xml", "zip"}
	var list []PHPExtension

	if runtime.GOOS == "windows" {
		for i, ext := range standardExts {
			list = append(list, PHPExtension{
				Name:    ext,
				Enabled: i%2 == 0,
			})
		}
		return list
	}

	cmd := exec.Command("docker", "exec", container, "php", "-m")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return list
	}

	modules := make(map[string]bool)
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(strings.ToLower(l))
		if l != "" && !strings.Contains(l, "[") {
			modules[l] = true
		}
	}

	for _, ext := range standardExts {
		enabled := false
		extLower := strings.ToLower(ext)
		if modules[extLower] {
			enabled = true
		} else {
			for m := range modules {
				if strings.Contains(m, extLower) {
					enabled = true
					break
				}
			}
		}

		list = append(list, PHPExtension{
			Name:    ext,
			Enabled: enabled,
		})
	}

	return list
}

func toggleContainerPHPExtension(container string, name string, enable bool) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	checkCmd := exec.Command("docker", "exec", container, "which", "docker-php-ext-enable")
	err := checkCmd.Run()
	if err == nil {
		action := "docker-php-ext-disable"
		if enable {
			action = "docker-php-ext-enable"
		}
		if enable {
			cmd := exec.Command("docker", "exec", container, action, name)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to enable extension: %s (%s)", err.Error(), string(out))
			}
			return nil
		} else {
			disableCmd := fmt.Sprintf("rm -f /usr/local/etc/php/conf.d/*%s*.ini", name)
			cmd := exec.Command("docker", "exec", container, "sh", "-c", disableCmd)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to disable extension: %s (%s)", err.Error(), string(out))
			}
			return nil
		}
	}

	checkCmd2 := exec.Command("docker", "exec", container, "which", "phpenmod")
	err2 := checkCmd2.Run()
	if err2 == nil {
		action := "phpdismod"
		if enable {
			action = "phpenmod"
		}
		cmd := exec.Command("docker", "exec", container, action, name)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to toggle extension via phpenmod: %s (%s)", err.Error(), string(out))
		}
		return nil
	}

	return fmt.Errorf("toggling extensions is not supported for this container. No standard php extension manager (docker-php-ext-enable/phpenmod) was found.")
}

type PHPSettings struct {
	MemoryLimit        string `json:"memory_limit"`
	UploadMaxFilesize  string `json:"upload_max_filesize"`
	PostMaxSize        string `json:"post_max_size"`
	MaxExecutionTime   string `json:"max_execution_time"`
	DisplayErrors      string `json:"display_errors"`
}

type PHPExtension struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func registerPHPRoutes(api *gin.RouterGroup) {
	// 1. Get installed PHP versions and statuses (including Docker containers)
	api.GET("/php/versions", func(c *gin.Context) {
		versions := getPHPVersions()
		dockerVersions := getDockerPHPVersions()
		versions = append(versions, dockerVersions...)
		c.JSON(200, versions)
	})

	// 2. Get settings for a specific PHP version or container
	api.GET("/php/settings", func(c *gin.Context) {
		version := c.Query("version")
		if version == "" {
			c.JSON(400, gin.H{"error": "Version or container ID is required"})
			return
		}
		isContainer := c.Query("is_container") == "true"

		if isContainer {
			if runtime.GOOS == "windows" {
				// Simulation settings for container
				settings := PHPSettings{
					MemoryLimit:        "256M",
					UploadMaxFilesize:  "50M",
					PostMaxSize:        "50M",
					MaxExecutionTime:   "120",
					DisplayErrors:      "On",
				}
				c.JSON(200, settings)
				return
			}

			// Verify if container is running
			cmdCheck := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", version)
			outCheck, errCheck := cmdCheck.Output()
			if errCheck != nil || strings.TrimSpace(string(outCheck)) != "true" {
				c.JSON(400, gin.H{"error": "Container is stopped. Please start the container first to manage its PHP settings."})
				return
			}

			// Get settings from running container
			cmd := exec.Command("docker", "exec", version, "php", "-r", "echo ini_get('memory_limit') . '|' . ini_get('upload_max_filesize') . '|' . ini_get('post_max_size') . '|' . ini_get('max_execution_time') . '|' . ini_get('display_errors');")
			out, err := cmd.Output()
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to read PHP settings from container: " + err.Error()})
				return
			}
			parts := strings.Split(strings.TrimSpace(string(out)), "|")
			if len(parts) < 5 {
				c.JSON(500, gin.H{"error": "Invalid response from container PHP: " + string(out)})
				return
			}
			displayErr := "Off"
			if parts[4] == "1" || strings.ToLower(parts[4]) == "on" || strings.ToLower(parts[4]) == "stdout" || strings.ToLower(parts[4]) == "stderr" {
				displayErr = "On"
			}
			settings := PHPSettings{
				MemoryLimit:        parts[0],
				UploadMaxFilesize:  parts[1],
				PostMaxSize:        parts[2],
				MaxExecutionTime:   parts[3],
				DisplayErrors:      displayErr,
			}
			c.JSON(200, settings)
			return
		}
		
		iniPath := getPHPMiniPath(version)
		if iniPath == "" {
			c.JSON(404, gin.H{"error": "PHP version or php.ini not found"})
			return
		}

		settings, err := loadPHPSettings(iniPath)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to read php.ini: " + err.Error()})
			return
		}

		c.JSON(200, settings)
	})

	// 3. Update settings for a specific PHP version or container
	api.POST("/php/settings", func(c *gin.Context) {
		var req struct {
			Version     string      `json:"version"`
			IsContainer bool        `json:"is_container"`
			Settings    PHPSettings `json:"settings"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		if req.IsContainer {
			if runtime.GOOS == "windows" {
				c.JSON(200, gin.H{"status": "ok", "message": "Simulation: Saved settings for container " + req.Version})
				return
			}

			// Verify if container is running
			cmdCheck := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", req.Version)
			outCheck, errCheck := cmdCheck.Output()
			if errCheck != nil || strings.TrimSpace(string(outCheck)) != "true" {
				c.JSON(400, gin.H{"error": "Container is stopped. Please start the container first to manage its PHP settings."})
				return
			}

			// Write custom configuration to /usr/local/etc/php/conf.d/uploads.ini (standard official path)
			// or find standard ini scan dir
			scanDir := findContainerIniScanDir(req.Version)
			var destPath string
			if scanDir != "" {
				destPath = filepath.ToSlash(filepath.Join(scanDir, "uploads.ini"))
			} else {
				loadedIni := findContainerLoadedIni(req.Version)
				if loadedIni != "" {
					destPath = loadedIni
				} else {
					destPath = "/usr/local/etc/php/conf.d/uploads.ini"
				}
			}

			iniContent := fmt.Sprintf("memory_limit = %s\nupload_max_filesize = %s\npost_max_size = %s\nmax_execution_time = %s\ndisplay_errors = %s\n",
				req.Settings.MemoryLimit, req.Settings.UploadMaxFilesize, req.Settings.PostMaxSize, req.Settings.MaxExecutionTime, req.Settings.DisplayErrors)

			writeCmd := exec.Command("docker", "exec", "-i", req.Version, "sh", "-c", fmt.Sprintf("mkdir -p %s && cat > %s", filepath.Dir(destPath), destPath))
			writeCmd.Stdin = strings.NewReader(iniContent)
			output, err := writeCmd.CombinedOutput()
			if err != nil {
				c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to write config into container: %s (%s)", err.Error(), string(output))})
				return
			}

			// Restart container to apply settings
			_ = exec.Command("docker", "restart", req.Version).Run()
			c.JSON(200, gin.H{"status": "ok"})
			return
		}

		iniPath := getPHPMiniPath(req.Version)
		if iniPath == "" {
			c.JSON(404, gin.H{"error": "php.ini not found"})
			return
		}

		err := savePHPSettings(iniPath, req.Settings)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to save settings: " + err.Error()})
			return
		}

		// Restart PHP-FPM Service
		if runtime.GOOS != "windows" {
			serviceName := fmt.Sprintf("php%s-fpm", req.Version)
			_ = exec.Command("systemctl", "restart", serviceName).Run()
		}

		c.JSON(200, gin.H{"status": "ok"})
	})

	// 4. Get extensions list for a specific PHP version or container
	api.GET("/php/extensions", func(c *gin.Context) {
		version := c.Query("version")
		if version == "" {
			c.JSON(400, gin.H{"error": "Version or container ID is required"})
			return
		}
		isContainer := c.Query("is_container") == "true"

		if isContainer {
			if runtime.GOOS != "windows" {
				// Verify if container is running
				cmdCheck := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", version)
				outCheck, errCheck := cmdCheck.Output()
				if errCheck != nil || strings.TrimSpace(string(outCheck)) != "true" {
					c.JSON(400, gin.H{"error": "Container is stopped. Please start the container first to manage its PHP extensions."})
					return
				}
			}
			exts := getContainerPHPExtensions(version)
			c.JSON(200, exts)
			return
		}

		exts := getPHPExtensions(version)
		c.JSON(200, exts)
	})

	// 5. Toggle extension state
	api.POST("/php/extensions/toggle", func(c *gin.Context) {
		var req struct {
			Version     string `json:"version"`
			IsContainer bool   `json:"is_container"`
			Name        string `json:"name"`
			Enable      bool   `json:"enable"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		if req.IsContainer {
			if runtime.GOOS != "windows" {
				// Verify if container is running
				cmdCheck := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", req.Version)
				outCheck, errCheck := cmdCheck.Output()
				if errCheck != nil || strings.TrimSpace(string(outCheck)) != "true" {
					c.JSON(400, gin.H{"error": "Container is stopped. Please start the container first to manage its PHP extensions."})
					return
				}
			}
			err := toggleContainerPHPExtension(req.Version, req.Name, req.Enable)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			// Restart container to apply
			_ = exec.Command("docker", "restart", req.Version).Run()
			c.JSON(200, gin.H{"status": "ok"})
			return
		}

		err := togglePHPExtension(req.Version, req.Name, req.Enable)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// Restart PHP-FPM
		if runtime.GOOS != "windows" {
			serviceName := fmt.Sprintf("php%s-fpm", req.Version)
			_ = exec.Command("systemctl", "restart", serviceName).Run()
		}

		c.JSON(200, gin.H{"status": "ok"})
	})
}

func getPHPVersions() []PHPVersionInfo {
	if runtime.GOOS == "windows" {
		return []PHPVersionInfo{
			{Version: "7.4", Status: "running"},
			{Version: "8.3", Status: "running"},
			{Version: "8.4", Status: "not_installed"},
		}
	}

	var list []PHPVersionInfo
	phpDir := "/etc/php"
	entries, err := os.ReadDir(phpDir)
	if err != nil {
		return list
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		ver := entry.Name()
		// Validate version name matches e.g. 7.4 or 8.1
		match, _ := regexp.MatchString(`^\d\.\d$`, ver)
		if !match {
			continue
		}

		// Check if FPM config exists
		fpmIni := filepath.Join(phpDir, ver, "fpm", "php.ini")
		if _, err := os.Stat(fpmIni); os.IsNotExist(err) {
			continue
		}

		status := "stopped"
		serviceName := fmt.Sprintf("php%s-fpm", ver)
		cmd := exec.Command("systemctl", "is-active", serviceName)
		out, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(out)) == "active" {
			status = "running"
		}

		list = append(list, PHPVersionInfo{
			Version: ver,
			Status:  status,
		})
	}

	return list
}

func getPHPMiniPath(version string) string {
	if runtime.GOOS == "windows" {
		// Simulation file
		simPath := filepath.Join(".", "data", fmt.Sprintf("php_%s_sim.ini", version))
		if _, err := os.Stat(simPath); os.IsNotExist(err) {
			_ = os.MkdirAll(filepath.Dir(simPath), 0755)
			defaultIni := "memory_limit = 128M\nupload_max_filesize = 20M\npost_max_size = 20M\nmax_execution_time = 30\ndisplay_errors = Off\n"
			_ = os.WriteFile(simPath, []byte(defaultIni), 0644)
		}
		return simPath
	}

	path := fmt.Sprintf("/etc/php/%s/fpm/php.ini", version)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func loadPHPSettings(iniPath string) (PHPSettings, error) {
	file, err := os.Open(iniPath)
	if err != nil {
		return PHPSettings{}, err
	}
	defer file.Close()

	settings := PHPSettings{
		MemoryLimit:       "128M",
		UploadMaxFilesize: "2M",
		PostMaxSize:       "8M",
		MaxExecutionTime:  "30",
		DisplayErrors:     "Off",
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Remove inline comments
		if idx := strings.Index(val, ";"); idx != -1 {
			val = strings.TrimSpace(val[:idx])
		}
		val = strings.Trim(val, `"'`)

		switch key {
		case "memory_limit":
			settings.MemoryLimit = val
		case "upload_max_filesize":
			settings.UploadMaxFilesize = val
		case "post_max_size":
			settings.PostMaxSize = val
		case "max_execution_time":
			settings.MaxExecutionTime = val
		case "display_errors":
			settings.DisplayErrors = val
		}
	}

	return settings, scanner.Err()
}

func savePHPSettings(iniPath string, settings PHPSettings) error {
	data, err := os.ReadFile(iniPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	keys := map[string]string{
		"memory_limit":        settings.MemoryLimit,
		"upload_max_filesize": settings.UploadMaxFilesize,
		"post_max_size":       settings.PostMaxSize,
		"max_execution_time":  settings.MaxExecutionTime,
		"display_errors":      settings.DisplayErrors,
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}

		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		if newVal, ok := keys[key]; ok {
			lines[i] = fmt.Sprintf("%s = %s", key, newVal)
			delete(keys, key) // Handled
		}
	}

	// Append any keys not found in file (should not happen for default ini)
	for key, val := range keys {
		lines = append(lines, fmt.Sprintf("%s = %s", key, val))
	}

	return os.WriteFile(iniPath, []byte(strings.Join(lines, "\n")), 0644)
}

func getPHPExtensions(version string) []PHPExtension {
	standardExts := []string{"opcache", "redis", "imagick", "gd", "pdo_mysql", "curl", "mbstring", "xml", "zip"}
	var list []PHPExtension

	if runtime.GOOS == "windows" {
		// Mock extension states
		for i, ext := range standardExts {
			list = append(list, PHPExtension{
				Name:    ext,
				Enabled: i%2 == 0,
			})
		}
		return list
	}

	modsDir := fmt.Sprintf("/etc/php/%s/mods-available", version)
	confDir := fmt.Sprintf("/etc/php/%s/fpm/conf.d", version)

	for _, ext := range standardExts {
		// Check if mod file is available
		modFile := filepath.Join(modsDir, ext+".ini")
		if _, err := os.Stat(modFile); os.IsNotExist(err) {
			// Fallback: check general file names in mods-available
			matches, _ := filepath.Glob(filepath.Join(modsDir, "*"+ext+"*"))
			if len(matches) == 0 {
				continue // Extension not available on OS
			}
		}

		// Check if active in fpm/conf.d
		enabled := false
		matches, _ := filepath.Glob(filepath.Join(confDir, "*"+ext+"*"))
		if len(matches) > 0 {
			enabled = true
		}

		list = append(list, PHPExtension{
			Name:    ext,
			Enabled: enabled,
		})
	}

	return list
}

func togglePHPExtension(version string, name string, enable bool) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	// We use php's official phpenmod / phpdismod commands
	action := "phpdismod"
	if enable {
		action = "phpenmod"
	}

	cmd := exec.Command(action, "-v", version, "-s", "fpm", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to %s extension %s: %s %w", action, name, string(output), err)
	}

	return nil
}
