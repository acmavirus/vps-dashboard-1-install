package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/disk"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

func main() {
	r := gin.Default()

	// 1. API
	r.GET("/api/stats", func(c *gin.Context) {
		vm, _ := mem.VirtualMemory()
		cpuPercent, _ := cpu.Percent(0, false)
		d, _ := disk.Usage("/")
		c.JSON(200, gin.H{
			"cpu":    cpuPercent[0],
			"ram":    vm.UsedPercent,
			"disk":   d.UsedPercent,
			"uptime": vm.Total,
		})
	})

	// 2. Serve Frontend
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
