package main

import (
	"embed"
	"flag"
	"fmt"
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
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

var Version = "v1.0.1"

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
	resp, err := http.PostForm(apiURL, url.Values{
		"chat_id": {chatID},
		"text":    {message},
	})
	if err != nil {
		log.Printf("[ERROR] Telegram alert failed: %v", err)
		return
	}
	defer resp.Body.Close()
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
		msg := fmt.Sprintf("🚨 [CPU ALERT] VPS: %s\nLoad is too high: %.1f%%", stats.Hostname, stats.CPU)
		go sendTelegram(msg)
		lastCpuAlert = time.Now()
	}

	if stats.Connections > 1500 && time.Since(lastDdosAlert) > 10*time.Minute {
		msg := fmt.Sprintf("⚠️ [DDoS ALERT] VPS: %s\nDetected %d TCP connections! Possible attack.", stats.Hostname, stats.Connections)
		go sendTelegram(msg)
		lastDdosAlert = time.Now()
	}

	return stats
}

func getLogs() (string, string) {
	logPath := "/var/log/syslog"
	if runtime.GOOS == "windows" {
		return "Log viewer only supports Linux.", "Windows Simulation"
	}
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		logPath = "/var/log/messages"
	}
	cmd := exec.Command("tail", "-n", "30", logPath)
	output, _ := cmd.CombinedOutput()
	return string(output), logPath
}

// Middleware xử lý CORS và Socket.io
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func main() {
	_ = godotenv.Load(".env")
	vFlag := flag.Bool("v", false, "Version")
	flag.Parse()
	if *vFlag {
		fmt.Printf("VPS Dashboard Version: %s\n", Version)
		return
	}

	r := gin.Default()
	r.Use(CorsMiddleware())

	// Cấu hình Socket.io với đầy đủ transports và CORS
	server := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			&polling.Transport{},
			&websocket.Transport{
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
			},
		},
	})

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		log.Println("connected:", s.ID())
		return nil
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		log.Println("socketio error:", e)
	})

	go func() {
		for {
			stats := getStats()
			server.BroadcastToNamespace("/", "stats", stats)
			logs, path := getLogs()
			server.BroadcastToNamespace("/", "logs", gin.H{"logs": logs, "path": path})
			time.Sleep(1 * time.Second)
		}
	}()

	go server.Serve()
	defer server.Close()

	r.GET("/socket.io/*any", gin.WrapH(server))
	r.POST("/socket.io/*any", gin.WrapH(server))

	r.GET("/api/stats", func(c *gin.Context) { c.JSON(200, getStats()) })
	r.GET("/api/logs", func(c *gin.Context) {
		logs, path := getLogs()
		c.JSON(200, gin.H{"logs": logs, "path": path})
	})

	publicFS, _ := fs.Sub(frontendFS, "frontend/dist")
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			c.JSON(404, gin.H{"error": "API route not found"})
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
		case strings.HasSuffix(trimPath, ".html"): contentType = "text/html"
		case strings.HasSuffix(trimPath, ".js"): contentType = "application/javascript"
		case strings.HasSuffix(trimPath, ".css"): contentType = "text/css"
		case strings.HasSuffix(trimPath, ".svg"): contentType = "image/svg+xml"
		case strings.HasSuffix(trimPath, ".png"): contentType = "image/png"
		case strings.HasSuffix(trimPath, ".ico"): contentType = "image/x-icon"
		}
		c.Data(200, contentType, data)
	})

	port := os.Getenv("PORT")
	if port == "" { port = "8900" }
	log.Printf("🚀 AcmaDash %s running on :%s\n", Version, port)
	r.Run(":" + port)
}
