package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

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
}

func main() {
	r := gin.Default()

	// 1. API Stats
	r.GET("/api/stats", func(c *gin.Context) {
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

		c.JSON(200, SystemStats{
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
		})
	})

	// 2. API Logs
	r.GET("/api/logs", func(c *gin.Context) {
		// Mặc định đọc syslog (dành cho Linux)
		logPath := "/var/log/syslog"
		if runtime.GOOS == "windows" {
			// Trên Windows, trả về thông báo giả lập cho test
			c.JSON(200, gin.H{
				"logs": "Hệ điều hành Windows: Chức năng xem log server thật sự chỉ áp dụng cho Linux.\nĐây là log giả lập để kiểm tra UI.",
			})
			return
		}

		// Kiểm tra file log tồn tại (Ubuntu/Debian mặ định dùng syslog)
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			logPath = "/var/log/messages" // CentOS/RedHat
		}

		// Lấy 100 dòng cuối bằng lệnh tail
		cmd := exec.Command("tail", "-n", "100", logPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(500, gin.H{"error": "Không thể đọc log: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"logs": string(output),
			"path": logPath,
		})
	})

	// 3. Serve Frontend
	publicFS, _ := fs.Sub(frontendFS, "frontend/dist")

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			c.JSON(404, gin.H{"error": "API route not found"})
			return
		}

		// Thử tìm file trong FS
		f, err := publicFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			c.FileFromFS(path, http.FS(publicFS))
			return
		}

		// Fallback về index.html (cho SPA routing)
		c.FileFromFS("index.html", http.FS(publicFS))
	})

	log.Println("Dashboard đang chạy tại http://localhost:8900")
	r.Run(":8900")
}
