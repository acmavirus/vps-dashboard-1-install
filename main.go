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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

var Version = "v1.1.1"

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
	return map[string]interface{}{
		"system": gin.H{
			"content": getTail("/var/log/syslog", 30),
			"path":    "/var/log/syslog",
		},
		"nginx_access": gin.H{
			"content": getTail("/var/log/nginx/access.log", 30),
			"path":    "/var/log/nginx/access.log",
		},
		"nginx_error": gin.H{
			"content": getTail("/var/log/nginx/error.log", 30),
			"path":    "/var/log/nginx/error.log",
		},
	}
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8900"
	}
	log.Printf("🚀 AcmaDash %s running on :%s\n", Version, port)
	r.Run(":" + port)
}
