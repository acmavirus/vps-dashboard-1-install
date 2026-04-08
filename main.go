package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

var Version = "v1.1.3"

//go:embed all:frontend/dist
var frontendFS embed.FS

var (
	lastCpuAlert  time.Time
	lastDdosAlert time.Time
)

type SystemStats struct {
	CPU          float64 `json:"cpu"`
	RAM          float64 `json:"ram"`
	RAMTotal     uint64  `json:"ram_total"`
	RAMUsed      uint64  `json:"ram_used"`
	Disk         float64 `json:"disk"`
	DiskTotal    uint64  `json:"disk_total"`
	DiskUsed     uint64  `json:"disk_used"`
	Uptime       uint64  `json:"uptime"`
	Hostname     string  `json:"hostname"`
	OS           string  `json:"os"`
	Platform     string  `json:"platform"`
	Kernel       string  `json:"kernel"`
	NetSent      uint64  `json:"net_sent"`
	NetRecv      uint64  `json:"net_recv"`
	Connections  int     `json:"connections"`
	Timestamp    int64   `json:"timestamp"`
	Version      string  `json:"version"`
}

type ProcessInfo struct {
	PID     int32   `json:"pid"`
	Name    string  `json:"name"`
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	Command string  `json:"command"`
}

type DockerInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Image  string `json:"image"`
	CPU    string `json:"cpu"`
	MEM    string `json:"mem"`
}

func sendTelegram(message string) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chatID == "" {
		return
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	_, _ = http.PostForm(apiURL, url.Values{
		"chat_id": {chatID},
		"text":    {message},
	})
}

func getStats() SystemStats {
	vm, _ := mem.VirtualMemory()
	cpuPercent, _ := cpu.Percent(0, false)
	d, _ := disk.Usage("/")
	h, _ := host.Info()
	n, _ := net.IOCounters(false)
	c, _ := net.Connections("tcp")

	var netSent, netRecv uint64
	if len(n) > 0 {
		netSent = n[0].BytesSent
		netRecv = n[0].BytesRecv
	}

	stats := SystemStats{
		CPU:         cpuPercent[0],
		RAM:         vm.UsedPercent,
		RAMTotal:    vm.Total,
		RAMUsed:     vm.Used,
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

	if len(results) > 10 {
		return results[:10]
	}
	return results
}

func getDockerStats() []DockerInfo {
	if runtime.GOOS == "windows" {
		return []DockerInfo{{Name: "demo-container", Status: "Running", Image: "nginx:latest", CPU: "0.5%", MEM: "120MB"}}
	}
	// Use docker stats command for simplicity
	cmd := exec.Command("docker", "stats", "--no-stream", "--format", "{{.Name}}|{{.CPUPerc}}|{{.MemUsage}}|{{.MemPerc}}")
	output, _ := cmd.CombinedOutput()
	lines := strings.Split(string(output), "\n")

	// Also get statuses
	cmdStat := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}|{{.Status}}|{{.Image}}")
	outStat, _ := cmdStat.CombinedOutput()
	statLines := strings.Split(string(outStat), "\n")
	statusMap := make(map[string]DockerInfo)
	for _, l := range statLines {
		parts := strings.Split(l, "|")
		if len(parts) >= 3 {
			statusMap[parts[0]] = DockerInfo{Name: parts[0], Status: parts[1], Image: parts[2]}
		}
	}

	var results []DockerInfo
	for _, l := range lines {
		parts := strings.Split(l, "|")
		if len(parts) >= 3 {
			name := parts[0]
			if info, ok := statusMap[name]; ok {
				info.CPU = parts[1]
				info.MEM = parts[2]
				results = append(results, info)
			}
		}
	}
	return results
}

func getTail(path string, lines int) string {
	if runtime.GOOS == "windows" {
		return "Log viewer only supports Linux (Simulation Mode)."
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Sprintf("File %s not found.", path)
	}
	cmd := exec.Command("tail", "-n", fmt.Sprintf("%d", lines), path)
	output, _ := cmd.CombinedOutput()
	return string(output)
}

func getAllLogs() map[string]interface{} {
	logs := map[string]interface{}{
		"system": gin.H{
			"content": getTail("/var/log/syslog", 30),
			"path":    "/var/log/syslog",
		},
	}

	// Real logic for Linux
	nginxDir := "/var/log/nginx/"
	if runtime.GOOS == "windows" {
		// Just for local dev visibility without crashing
		nginxDir = "./logs/nginx/" 
		_ = os.MkdirAll(nginxDir, 0755)
	}
	files, err := os.ReadDir(nginxDir)
	if err != nil {
		logs["nginx_error"] = gin.H{"content": fmt.Sprintf("Error reading %s: %v", nginxDir, err), "path": nginxDir}
		return logs
	}

	sitesMap := make(map[string]map[string]gin.H)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		path := nginxDir + name

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
		} else if name == "access.log" || name == "error.log" {
			// Standard logs
			key := "nginx_access"
			if name == "error.log" {
				key = "nginx_error"
			}
			logs[key] = gin.H{"content": getTail(path, 30), "path": path}
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

func main() {
	_ = godotenv.Load(".env")
	vFlag := flag.Bool("v", false, "Version")
	flag.Parse()
	if *vFlag {
		fmt.Printf("Version: %s\n", Version)
		return
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 1. API - Standard
	r.GET("/api/stats", func(c *gin.Context) {
		c.JSON(200, getStats())
	})

	r.GET("/api/logs", func(c *gin.Context) {
		c.JSON(200, getAllLogs())
	})

	// 2. API - Live Streaming (SSE)
	r.GET("/api/stream", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

		ticker := time.NewTicker(1 * time.Second) // 1s update frequency for real-time feel
		defer ticker.Stop()

		c.Stream(func(w io.Writer) bool {
			select {
			case <-ticker.C:
				stats := getStats()
				logs := getAllLogs()
				data, _ := json.Marshal(gin.H{
					"stats": stats,
					"logs":  logs,
				})
				c.SSEvent("message", string(data))
				return true
			case <-c.Request.Context().Done():
				return false
			}
		})
	})

	// 3. Static Files
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

	// New API endpoints
	r.GET("/api/processes", func(c *gin.Context) {
		c.JSON(200, getTopProcesses())
	})

	r.GET("/api/docker", func(c *gin.Context) {
		c.JSON(200, getDockerStats())
	})

	r.POST("/api/control", func(c *gin.Context) {
		var req struct {
			Service string `json:"service"`
			Action  string `json:"action"` // start, stop, restart
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		
		// Map service names to systemd services
		services := map[string]string{
			"nginx": "nginx",
			"php8.3": "php8.3-fpm",
			"php7.4": "php7.4-fpm",
			"mysql": "mariadb",
		}
		
		target, ok := services[req.Service]
		if !ok {
			c.JSON(400, gin.H{"error": "Service not allowed"})
			return
		}

		cmd := exec.Command("systemctl", req.Action, target)
		err := cmd.Run()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8900"
	}
	log.Printf("🚀 AcmaDash %s running on :%s\n", Version, port)
	r.Run(":" + port)
}
