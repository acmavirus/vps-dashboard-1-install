package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func registerCronRoutes(api *gin.RouterGroup) {
	api.GET("/cron", func(c *gin.Context) {
		jobs, err := listCronJobs()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, jobs)
	})

	api.POST("/cron/add", func(c *gin.Context) {
		var req struct {
			Name     string `json:"name"`
			Schedule string `json:"schedule"`
			Command  string `json:"command"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		if req.Name == "" || req.Schedule == "" || req.Command == "" {
			c.JSON(400, gin.H{"error": "All fields are required"})
			return
		}

		jobs, err := listCronJobs()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		logDir := "/var/log/cron_tasks"
		_ = os.MkdirAll(logDir, 0755)

		id := fmt.Sprintf("cron_%d", time.Now().UnixNano())
		logPath := fmt.Sprintf("%s/%s.log", logDir, id)

		newJob := CronJob{
			ID:       id,
			Name:     req.Name,
			Schedule: req.Schedule,
			Command:  req.Command,
			Status:   "enabled",
			LogPath:  logPath,
			IsSystem: false,
		}

		jobs = append(jobs, newJob)
		if err := saveCronJobs(jobs); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "ok"})
	})

	api.POST("/cron/delete", func(c *gin.Context) {
		var req struct {
			ID string `json:"id"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		jobs, err := listCronJobs()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		var remainingJobs []CronJob
		found := false
		for _, job := range jobs {
			if job.ID == req.ID {
				found = true
				if job.LogPath != "" {
					_ = os.Remove(job.LogPath)
				}
				continue
			}
			remainingJobs = append(remainingJobs, job)
		}

		if !found {
			c.JSON(404, gin.H{"error": "Cronjob not found"})
			return
		}

		if err := saveCronJobs(remainingJobs); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "ok"})
	})

	api.POST("/cron/toggle", func(c *gin.Context) {
		var req struct {
			ID string `json:"id"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		jobs, err := listCronJobs()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		found := false
		for i, job := range jobs {
			if job.ID == req.ID {
				found = true
				if job.Status == "enabled" {
					jobs[i].Status = "disabled"
				} else {
					jobs[i].Status = "enabled"
				}
				break
			}
		}

		if !found {
			c.JSON(404, gin.H{"error": "Cronjob not found"})
			return
		}

		if err := saveCronJobs(jobs); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "ok"})
	})

	api.GET("/cron/log", func(c *gin.Context) {
		id := c.Query("id")
		if id == "" {
			c.JSON(400, gin.H{"error": "ID is required"})
			return
		}

		logPath := fmt.Sprintf("/var/log/cron_tasks/%s.log", id)
		data, err := os.ReadFile(logPath)
		if err != nil {
			if os.IsNotExist(err) {
				c.JSON(200, gin.H{"log": "[No logs generated yet. Executing task will write to this file.]"})
				return
			}
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"log": string(data)})
	})

	api.POST("/cron/log/clear", func(c *gin.Context) {
		var req struct {
			ID string `json:"id"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		logPath := fmt.Sprintf("/var/log/cron_tasks/%s.log", req.ID)
		_ = os.WriteFile(logPath, []byte(""), 0644)
		c.JSON(200, gin.H{"status": "ok"})
	})
}

func listCronJobs() ([]CronJob, error) {
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "no crontab") {
		return nil, fmt.Errorf("failed to run crontab -l: %s %w", string(output), err)
	}

	jobs := []CronJob{}
	lines := strings.Split(string(output), "\n")
	var currentJob *CronJob

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "# PANELD_ID:") {
			if currentJob != nil {
				jobs = append(jobs, *currentJob)
			}
			currentJob = &CronJob{
				ID:       strings.TrimSpace(strings.TrimPrefix(trimmed, "# PANELD_ID:")),
				Status:   "enabled",
				IsSystem: false,
			}
			continue
		}

		if currentJob != nil {
			if strings.HasPrefix(trimmed, "# PANELD_NAME:") {
				currentJob.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "# PANELD_NAME:"))
				continue
			}
			if strings.HasPrefix(trimmed, "# PANELD_STATUS:") {
				currentJob.Status = strings.TrimSpace(strings.TrimPrefix(trimmed, "# PANELD_STATUS:"))
				continue
			}
			if strings.HasPrefix(trimmed, "# PANELD_LOG:") {
				currentJob.LogPath = strings.TrimSpace(strings.TrimPrefix(trimmed, "# PANELD_LOG:"))
				continue
			}
		}

		if currentJob != nil && !strings.HasPrefix(trimmed, "# PANELD_") {
			cronLine := trimmed
			if currentJob.Status == "disabled" && strings.HasPrefix(cronLine, "#") {
				cronLine = strings.TrimSpace(strings.TrimPrefix(cronLine, "#"))
			}

			parts := strings.Fields(cronLine)
			if len(parts) >= 6 {
				currentJob.Schedule = strings.Join(parts[0:5], " ")
				fullCommand := strings.Join(parts[5:], " ")
				redirIndex := strings.Index(fullCommand, " > ")
				if redirIndex != -1 {
					currentJob.Command = strings.TrimSpace(fullCommand[:redirIndex])
				} else {
					currentJob.Command = fullCommand
				}
			} else {
				currentJob.Command = cronLine
			}
			jobs = append(jobs, *currentJob)
			currentJob = nil
			continue
		}

		if currentJob == nil && !strings.HasPrefix(trimmed, "# PANELD_") {
			if strings.HasPrefix(trimmed, "#") {
				continue
			}
			parts := strings.Fields(trimmed)
			if len(parts) >= 6 {
				schedule := strings.Join(parts[0:5], " ")
				cmd := strings.Join(parts[5:], " ")
				jobs = append(jobs, CronJob{
					ID:       "system_" + fmt.Sprintf("%d", len(jobs)),
					Name:     "System Job",
					Schedule: schedule,
					Command:  cmd,
					Status:   "enabled",
					IsSystem: true,
				})
			}
		}
	}

	if currentJob != nil {
		jobs = append(jobs, *currentJob)
	}

	return jobs, nil
}

func saveCronJobs(jobs []CronJob) error {
	var lines []string
	for _, job := range jobs {
		if job.IsSystem {
			lines = append(lines, fmt.Sprintf("%s %s", job.Schedule, job.Command))
		} else {
			lines = append(lines, fmt.Sprintf("# PANELD_ID: %s", job.ID))
			lines = append(lines, fmt.Sprintf("# PANELD_NAME: %s", job.Name))
			lines = append(lines, fmt.Sprintf("# PANELD_STATUS: %s", job.Status))
			if job.LogPath != "" {
				lines = append(lines, fmt.Sprintf("# PANELD_LOG: %s", job.LogPath))
			}
			
			cmdLine := job.Command
			if job.LogPath != "" {
				cmdLine = fmt.Sprintf("%s > %s 2>&1", job.Command, job.LogPath)
			}
			
			if job.Status == "disabled" {
				lines = append(lines, fmt.Sprintf("# %s %s", job.Schedule, cmdLine))
			} else {
				lines = append(lines, fmt.Sprintf("%s %s", job.Schedule, cmdLine))
			}
		}
		lines = append(lines, "")
	}

	tmpFile := "/tmp/new_crontab"
	err := os.WriteFile(tmpFile, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	output, err := exec.Command("crontab", tmpFile).CombinedOutput()
	if err != nil {
		return fmt.Errorf("crontab failed: %s %w", string(output), err)
	}

	return nil
}
