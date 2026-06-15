package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

func registerDockerRoutes(api *gin.RouterGroup) {
	api.GET("/docker", func(c *gin.Context) {
		c.JSON(200, getDockerStats())
	})

	api.POST("/docker/control", func(c *gin.Context) {
		var req struct {
			ID     string `json:"id"`
			Action string `json:"action"` // start, stop, restart, remove
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		idPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
		if !idPattern.MatchString(req.ID) {
			c.JSON(400, gin.H{"error": "Invalid container ID"})
			return
		}

		validActions := map[string]bool{"start": true, "stop": true, "restart": true, "remove": true}
		if !validActions[req.Action] {
			c.JSON(400, gin.H{"error": "Invalid action"})
			return
		}

		if runtime.GOOS == "windows" {
			c.JSON(200, gin.H{"status": "ok", "message": fmt.Sprintf("Simulation: %s container %s", req.Action, req.ID)})
			return
		}

		var cmd *exec.Cmd
		if req.Action == "remove" {
			cmd = exec.Command("docker", "rm", "-f", req.ID)
		} else {
			cmd = exec.Command("docker", req.Action, req.ID)
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error(), "details": string(output)})
			return
		}

		if req.Action == "remove" {
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
		}

		c.JSON(200, gin.H{"status": "ok"})
	})
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
