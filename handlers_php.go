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

	"github.com/gin-gonic/gin"
)

type PHPVersionInfo struct {
	Version string `json:"version"`
	Status  string `json:"status"` // running, stopped, not_installed
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
	// 1. Get installed PHP versions and statuses
	api.GET("/php/versions", func(c *gin.Context) {
		versions := getPHPVersions()
		c.JSON(200, versions)
	})

	// 2. Get settings for a specific PHP version
	api.GET("/php/settings", func(c *gin.Context) {
		version := c.Query("version")
		if version == "" {
			c.JSON(400, gin.H{"error": "Version is required"})
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

	// 3. Update settings for a specific PHP version
	api.POST("/php/settings", func(c *gin.Context) {
		var req struct {
			Version  string      `json:"version"`
			Settings PHPSettings `json:"settings"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
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

	// 4. Get extensions list for a specific PHP version
	api.GET("/php/extensions", func(c *gin.Context) {
		version := c.Query("version")
		if version == "" {
			c.JSON(400, gin.H{"error": "Version is required"})
			return
		}

		exts := getPHPExtensions(version)
		c.JSON(200, exts)
	})

	// 5. Toggle extension state
	api.POST("/php/extensions/toggle", func(c *gin.Context) {
		var req struct {
			Version string `json:"version"`
			Name    string `json:"name"`
			Enable  bool   `json:"enable"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
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
