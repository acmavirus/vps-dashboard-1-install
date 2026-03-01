package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// Phiên bản ứng dụng (Sẽ được cập nhật khi build release)
var Version = "v1.0.0"

//go:embed all:frontend/dist
var frontendFS embed.FS

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
	Timestamp    int64   `json:"timestamp"`
	Version      string  `json:"version"`
}

func getStats() SystemStats {
	vm, _ := mem.VirtualMemory()
	cpuPercent, _ := cpu.Percent(0, false)
	d, _ := disk.Usage("/")
	h, _ := host.Info()
	n, _ := net.IOCounters(false)

	var netSent, netRecv uint64
	if len(n) > 0 {
		netSent = n[0].BytesSent
		netRecv = n[0].BytesRecv
	}

	return SystemStats{
		CPU:       cpuPercent[0],
		RAM:       vm.UsedPercent,
		RAMTotal:  vm.Total,
		RAMUsed:   vm.Used,
		Disk:      d.UsedPercent,
		DiskTotal: d.Total,
		DiskUsed:  d.Used,
		Uptime:    h.Uptime,
		Hostname:  h.Hostname,
		OS:        runtime.GOOS,
		Platform:  h.Platform,
		Kernel:    h.KernelVersion,
		NetSent:   netSent,
		NetRecv:   netRecv,
		Timestamp: time.Now().Unix(),
		Version:   Version,
	}
}

func getLogs() (string, string) {
	logPath := "/var/log/syslog"
	if runtime.GOOS == "windows" {
		return "Trình xem log real-time chỉ hỗ trợ Linux.\nĐạt trạng thái: " + time.Now().Format("15:04:05") + " - Waiting for data...", "Windows Simulation"
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		logPath = "/var/log/messages"
	}

	cmd := exec.Command("tail", "-n", "30", logPath)
	output, _ := cmd.CombinedOutput()
	return string(output), logPath
}

func main() {
	vFlag := flag.Bool("v", false, "Hiển thị phiên bản")
	flag.Parse()

	if *vFlag {
		fmt.Printf("VPS Dashboard Version: %s\n", Version)
		return
	}

	r := gin.Default()

	// Socket.io setup
	server := socketio.NewServer(nil)

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		return nil
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		log.Println("meet error:", e)
	})

	// Background loop to push stats and logs
	go func() {
		for {
			stats := getStats()
			server.BroadcastToNamespace("/", "stats", stats)

			logs, path := getLogs()
			server.BroadcastToNamespace("/", "logs", gin.H{
				"logs": logs,
				"path": path,
			})

			time.Sleep(1 * time.Second)
		}
	}()

	go server.Serve()
	defer server.Close()

	// 1. HTTP Routes
	r.GET("/socket.io/*any", gin.WrapH(server))
	r.POST("/socket.io/*any", gin.WrapH(server))

	r.GET("/api/stats", func(c *gin.Context) {
		c.JSON(200, getStats())
	})

	r.GET("/api/logs", func(c *gin.Context) {
		logs, path := getLogs()
		c.JSON(200, gin.H{"logs": logs, "path": path})
	})

	// 2. Serve Frontend
	publicFS, _ := fs.Sub(frontendFS, "frontend/dist")

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			c.JSON(404, gin.H{"error": "API route not found"})
			return
		}

		f, err := publicFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			c.FileFromFS(path, http.FS(publicFS))
			return
		}

		c.FileFromFS("index.html", http.FS(publicFS))
	})

	log.Printf("AcmaDash %s đang chạy tại http://0.0.0.0:8900\n", Version)
	r.Run(":8900")
}
