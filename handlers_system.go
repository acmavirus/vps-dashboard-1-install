package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

func registerSystemRoutes(api *gin.RouterGroup) {
	api.GET("/stats", func(c *gin.Context) {
		c.JSON(200, getStats())
	})

	api.GET("/logs", func(c *gin.Context) {
		c.JSON(200, getAllLogs())
	})

	api.GET("/processes", func(c *gin.Context) {
		c.JSON(200, getTopProcesses())
	})

	api.POST("/processes/kill", func(c *gin.Context) {
		var req struct {
			PID int32 `json:"pid"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		if req.PID <= 1 {
			c.JSON(400, gin.H{"error": "Cannot kill system processes"})
			return
		}

		if req.PID == int32(os.Getpid()) {
			c.JSON(400, gin.H{"error": "Cannot kill the panel process itself"})
			return
		}

		if runtime.GOOS == "windows" {
			c.JSON(200, gin.H{"status": "ok", "message": fmt.Sprintf("Simulation: Killed PID %d", req.PID)})
			return
		}

		proc, err := process.NewProcess(req.PID)
		if err != nil {
			c.JSON(400, gin.H{"error": "Process not found: " + err.Error()})
			return
		}

		name, _ := proc.Name()
		name = strings.ToLower(name)
		systemNames := []string{"systemd", "init", "sshd", "dbus-daemon", "cron", "rsyslogd", "udevd"}
		for _, sysName := range systemNames {
			if strings.Contains(name, sysName) {
				c.JSON(400, gin.H{"error": "Cannot kill critical system process: " + name})
				return
			}
		}

		err = proc.Kill()
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to kill process: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "ok"})
	})

	api.GET("/pm2", func(c *gin.Context) {
		c.JSON(200, getPM2Stats())
	})

	// SSE - Real-time Streaming (Optimized: only stats, every 2s, no logs)
	api.GET("/stream", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		c.Stream(func(w io.Writer) bool {
			select {
			case <-ticker.C:
				stats := getStats()
				data, _ := json.Marshal(gin.H{
					"stats": stats,
				})
				c.SSEvent("message", string(data))
				return true
			case <-c.Request.Context().Done():
				return false
			}
		})
	})

	// --- Settings API Endpoints ---
	api.GET("/settings", func(c *gin.Context) {
		tokenVal := getSetting("telegram_bot_token", os.Getenv("TELEGRAM_BOT_TOKEN"))
		chatIDVal := getSetting("telegram_chat_id", os.Getenv("TELEGRAM_CHAT_ID"))
		c.JSON(200, gin.H{
			"username":           adminUser,
			"version":            Version,
			"go_version":         runtime.Version(),
			"os":                 runtime.GOOS + "/" + runtime.GOARCH,
			"num_cpu":            runtime.NumCPU(),
			"goroutines":         runtime.NumGoroutine(),
			"telegram_bot_token": tokenVal,
			"telegram_chat_id":   chatIDVal,
		})
	})

	api.POST("/settings/update", func(c *gin.Context) {
		var req struct {
			Username         string `json:"username"`
			Password         string `json:"password"`
			TelegramBotToken string `json:"telegram_bot_token"`
			TelegramChatID   string `json:"telegram_chat_id"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		if req.Username == "" {
			c.JSON(400, gin.H{"error": "Username cannot be empty"})
			return
		}

		if req.Password != "" && len(req.Password) < 6 {
			c.JSON(400, gin.H{"error": "Password must be at least 6 characters"})
			return
		}

		adminUser = req.Username
		if req.Password != "" {
			adminPass = req.Password
		}

		_ = saveSetting("admin_user", adminUser)
		_ = saveSetting("admin_pass", adminPass)
		_ = saveSetting("telegram_bot_token", req.TelegramBotToken)
		_ = saveSetting("telegram_chat_id", req.TelegramChatID)

		if err := saveSettingsToEnv(adminUser, adminPass); err != nil {
			c.JSON(500, gin.H{"error": "Failed to save settings: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "ok"})
	})

	api.POST("/settings/restart", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
		go func() {
			time.Sleep(1 * time.Second)
			_ = exec.Command("systemctl", "restart", "vps-dashboard").Run()
			os.Exit(0)
		}()
	})

	api.GET("/software", func(c *gin.Context) {
		if cachedSoftware == nil || time.Since(lastSoftwareCheck) > 60*time.Second {
			cachedSoftware = getSoftwareVersions()
			lastSoftwareCheck = time.Now()
		}
		c.JSON(200, cachedSoftware)
	})

	api.GET("/metrics/history", func(c *gin.Context) {
		limitStr := c.DefaultQuery("limit", "288")
		var limit int
		_, _ = fmt.Sscanf(limitStr, "%d", &limit)
		if limit <= 0 {
			limit = 288
		}
		history, err := getMetricsHistorySQL(limit)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, history)
	})
}

func saveSettingsToEnv(username, password string) error {
	var lines []string
	if data, err := os.ReadFile(".env"); err == nil {
		oldLines := strings.Split(string(data), "\n")
		for _, line := range oldLines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				if key == "ADMIN_USER" || key == "ADMIN_PASS" || key == "AUTH_TOKEN" {
					continue
				}
				lines = append(lines, line)
			}
		}
	}

	lines = append(lines, fmt.Sprintf("ADMIN_USER=%s", username))
	lines = append(lines, fmt.Sprintf("ADMIN_PASS=%s", password))
	lines = append(lines, fmt.Sprintf("AUTH_TOKEN=%s", authToken))

	return os.WriteFile(".env", []byte(strings.Join(lines, "\n")), 0600)
}

func getStats() SystemStats {
	vm, _ := mem.VirtualMemory()
	var cpuVal float64
	if cpuPercent, err := cpu.Percent(100*time.Millisecond, false); err == nil && len(cpuPercent) > 0 {
		cpuVal = cpuPercent[0]
	}
	d, _ := disk.Usage("/")
	h, _ := host.Info()
	n, _ := net.IOCounters(false)
	c, _ := net.Connections("tcp")
	swap, _ := mem.SwapMemory()

	var netSent, netRecv uint64
	if len(n) > 0 {
		netSent = n[0].BytesSent
		netRecv = n[0].BytesRecv
	}

	var load1, load5, load15 float64
	if runtime.GOOS != "windows" {
		if l, err := load.Avg(); err == nil {
			load1 = l.Load1
			load5 = l.Load5
			load15 = l.Load15
		}
	}

	dIO, _ := disk.IOCounters()
	var diskRead, diskWrite uint64
	for _, io := range dIO {
		diskRead += io.ReadBytes
		diskWrite += io.WriteBytes
	}

	spamAlertsMutex.RLock()
	alertsCopy := make([]SpamAlert, len(ActiveSpamAlerts))
	copy(alertsCopy, ActiveSpamAlerts)
	spamAlertsMutex.RUnlock()

	stats := SystemStats{
		CPU:         cpuVal,
		RAM:         vm.UsedPercent,
		RAMTotal:    vm.Total,
		RAMUsed:     vm.Used,
		SwapTotal:   swap.Total,
		SwapUsed:    swap.Used,
		SwapPercent: swap.UsedPercent,
		Disk:        d.UsedPercent,
		DiskTotal:   d.Total,
		DiskUsed:    d.Used,
		Uptime:      h.Uptime,
		Hostname:    h.Hostname,
		OS:          runtime.GOOS,
		Platform:    h.Platform,
		Kernel:      h.KernelVersion,
		NetSent:     netSent,
		NetRecv:     netRecv,
		Connections: len(c),
		Timestamp:   time.Now().Unix(),
		Version:     Version,
		Load1:       load1,
		Load5:       load5,
		Load15:      load15,
		CPUCores:    cachedCPUCores,
		CPUModel:    cachedCPUModel,
		DiskRead:    diskRead,
		DiskWrite:   diskWrite,
		SpamAlerts:  alertsCopy,
	}

	if stats.CPU > 90.0 && time.Since(lastCpuAlert) > 5*time.Minute {
		msg := fmt.Sprintf("🚨 [CPU ALERT] VPS: %s\nLoad: %.1f%%", stats.Hostname, stats.CPU)
		go sendTelegram(msg)
		lastCpuAlert = time.Now()
	}

	if stats.Connections > 2000 && time.Since(lastDdosAlert) > 10*time.Minute {
		msg := fmt.Sprintf("⚠️ [DDoS ALERT] VPS: %s\nConnections: %d", stats.Hostname, stats.Connections)
		go sendTelegram(msg)
		lastDdosAlert = time.Now()
	}

	return stats
}

func getTopProcesses() []ProcessInfo {
	processes, err := process.Processes()
	if err != nil {
		return nil
	}

	var results []ProcessInfo
	for _, p := range processes {
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()
		name, _ := p.Name()
		cmd, _ := p.Cmdline()
		if cpu > 0.1 || mem > 0.1 {
			results = append(results, ProcessInfo{
				PID:     p.Pid,
				Name:    name,
				CPU:     cpu,
				Memory:  float64(mem),
				Command: cmd,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CPU > results[j].CPU
	})

	if len(results) > 15 {
		return results[:15]
	}
	return results
}

func getPM2Stats() interface{} {
	if runtime.GOOS == "windows" {
		return []map[string]interface{}{
			{"name": "demo-api", "pm_id": 0, "status": "online", "monit": map[string]interface{}{"cpu": 1.2, "memory": 45000000}, "pm2_env": map[string]interface{}{"pm_uptime": time.Now().Unix() * 1000}},
		}
	}
	cmd := exec.Command("pm2", "jlist")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return []interface{}{}
	}
	var data interface{}
	_ = json.Unmarshal(output, &data)
	return data
}

func getTail(path string, lines int) string {
	if runtime.GOOS == "windows" {
		return "[Tail log simulation under Windows OS]"
	}
	cmd := exec.Command("tail", "-n", fmt.Sprintf("%d", lines), path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Failed to read log file: " + err.Error()
	}
	return string(output)
}

func getPanelLogs(lines int) string {
	if runtime.GOOS == "windows" {
		return "[Panel log simulation under Windows OS]"
	}
	cmd := exec.Command("journalctl", "-u", "vps-dashboard", "-n", fmt.Sprintf("%d", lines), "--no-pager")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Failed to fetch panel logs: " + err.Error()
	}
	return string(output)
}

func getAllLogs() map[string]interface{} {
	logs := map[string]interface{}{
		"system": gin.H{
			"content": getTail("/var/log/syslog", 30),
			"path":    "/var/log/syslog",
		},
		"panel": gin.H{
			"content": getPanelLogs(100),
			"path":    "journalctl -u vps-dashboard -n 100",
		},
	}

	paths := getDomainPaths()
	nginxDir := paths.nginxLogDir + string(filepath.Separator)
	sitesEnabledDir := paths.sitesEnabledDir

	if runtime.GOOS == "windows" {
		_ = os.MkdirAll(nginxDir, 0755)
		_ = os.MkdirAll(sitesEnabledDir, 0755)
	}

	// 1. Get domains from sites-enabled (the source of truth)
	domains := []string{}
	siteFiles, err := os.ReadDir(sitesEnabledDir)
	if err == nil {
		for _, f := range siteFiles {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			if name == "default" || name == "phpmyadmin" {
				continue
			}
			domain := strings.TrimSuffix(name, ".conf")
			domains = append(domains, domain)
		}
	}

	// 2. Also check log directory for other potential logs (fallback)
	logFiles, _ := os.ReadDir(nginxDir)
	sitesMap := make(map[string]map[string]gin.H)

	for _, d := range domains {
		sitesMap[d] = make(map[string]gin.H)
		accPath := nginxDir + d + "_access.log"
		errPath := nginxDir + d + "_error.log"
		ensureLogFileExists(accPath)
		ensureLogFileExists(errPath)
		sitesMap[d]["access"] = gin.H{"content": getTail(accPath, 30), "path": accPath}
		sitesMap[d]["error"] = gin.H{"content": getTail(errPath, 30), "path": errPath}
	}

	// Scan log directory to catch any missed or differently named logs
	if logFiles != nil {
		for _, f := range logFiles {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			path := nginxDir + name

			if name == "access.log" || name == "error.log" {
				key := "nginx_access"
				if name == "error.log" {
					key = "nginx_error"
				}
				logs[key] = gin.H{"content": getTail(path, 30), "path": path}
				continue
			}

			if strings.HasSuffix(name, "_access.log") {
				domain := strings.TrimSuffix(name, "_access.log")
				if _, ok := sitesMap[domain]; !ok {
					sitesMap[domain] = make(map[string]gin.H)
				}
				sitesMap[domain]["access"] = gin.H{"content": getTail(path, 30), "path": path}
			} else if strings.HasSuffix(name, "_error.log") {
				domain := strings.TrimSuffix(name, "_error.log")
				if _, ok := sitesMap[domain]; !ok {
					sitesMap[domain] = make(map[string]gin.H)
				}
				sitesMap[domain]["error"] = gin.H{"content": getTail(path, 30), "path": path}
			}
		}
	}

	var nginxSites []gin.H
	for domain, data := range sitesMap {
		site := gin.H{"domain": domain}
		if acc, ok := data["access"]; ok {
			site["access"] = acc
		}
		if err, ok := data["error"]; ok {
			site["error"] = err
		}
		nginxSites = append(nginxSites, site)
	}

	if len(nginxSites) > 0 {
		sort.Slice(nginxSites, func(i, j int) bool {
			return nginxSites[i]["domain"].(string) < nginxSites[j]["domain"].(string)
		})
		logs["nginx_sites"] = nginxSites
	}

	return logs
}

func getSoftwareVersions() SoftwareInfo {
	if runtime.GOOS == "windows" {
		return SoftwareInfo{
			Nginx:   "nginx/1.24.0 (Windows Sim)",
			PHP83:   "PHP 8.3.12 (Windows Sim)",
			PHP74:   "PHP 7.4.33 (Windows Sim)",
			MySQL:   "mysql Ver 15.1 Distrib 10.11.6-MariaDB (Windows Sim)",
			Redis:   "Redis server v=7.2.4 (Windows Sim)",
		}
	}

	var info SoftwareInfo

	// Nginx
	cmd := exec.Command("nginx", "-v")
	out, err := cmd.CombinedOutput() // nginx outputs version to stderr
	if err == nil {
		info.Nginx = parseVersionString(string(out))
	} else {
		info.Nginx = "Not Installed"
	}

	// PHP 8.3
	cmd = exec.Command("php8.3", "-v")
	out, err = cmd.Output()
	if err == nil {
		info.PHP83 = parseVersionString(string(out))
	} else {
		info.PHP83 = "Not Installed"
	}

	// PHP 7.4
	cmd = exec.Command("php7.4", "-v")
	out, err = cmd.Output()
	if err == nil {
		info.PHP74 = parseVersionString(string(out))
	} else {
		info.PHP74 = "Not Installed"
	}

	// MySQL/MariaDB
	cmd = exec.Command("mysql", "-V")
	out, err = cmd.Output()
	if err == nil {
		info.MySQL = parseVersionString(string(out))
	} else {
		info.MySQL = "Not Installed"
	}

	// Redis
	cmd = exec.Command("redis-server", "--version")
	out, err = cmd.Output()
	if err == nil {
		info.Redis = parseVersionString(string(out))
	} else {
		info.Redis = "Not Installed"
	}

	return info
}

func parseVersionString(raw string) string {
	raw = strings.TrimSpace(raw)
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return "Unknown"
	}
	firstLine := lines[0]
	if len(firstLine) > 60 {
		return firstLine[:60] + "..."
	}
	return firstLine
}

func startHistoricalMetricsCollector() {
	log.Println("📊 Starting Historical Metrics Collector worker...")
	
	// Record initially
	stats := getStats()
	_ = logSystemMetricsSQL(stats)
	_ = cleanOldMetricsSQL()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		stats := getStats()
		_ = logSystemMetricsSQL(stats)
		_ = cleanOldMetricsSQL()
	}
}
