package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

func registerFilesRoutes(api *gin.RouterGroup) {
	api.GET("/files", func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			path = "/"
		}
		path = filepath.Clean(path)

		info, err := os.Stat(path)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if !info.IsDir() {
			c.JSON(400, gin.H{"error": "Path is not a directory"})
			return
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		var files []FileInfo
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			files = append(files, FileInfo{
				Name:    entry.Name(),
				Size:    info.Size(),
				IsDir:   entry.IsDir(),
				ModTime: info.ModTime(),
				Mode:    info.Mode().String(),
			})
		}

		sort.Slice(files, func(i, j int) bool {
			if files[i].IsDir && !files[j].IsDir {
				return true
			}
			if !files[i].IsDir && files[j].IsDir {
				return false
			}
			return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
		})

		c.JSON(200, gin.H{
			"current_path": path,
			"files":        files,
		})
	})

	api.GET("/files/read", func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			c.JSON(400, gin.H{"error": "Path is required"})
			return
		}
		path = filepath.Clean(path)

		info, err := os.Stat(path)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if info.IsDir() {
			c.JSON(400, gin.H{"error": "Cannot read directory as text file"})
			return
		}
		if info.Size() > 5*1024*1024 {
			c.JSON(400, gin.H{"error": "File size exceeds 5MB limit for editor"})
			return
		}

		content, err := os.ReadFile(path)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"path":    path,
			"content": string(content),
		})
	})

	api.POST("/files/write", func(c *gin.Context) {
		var req struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		path := filepath.Clean(req.Path)

		err := os.WriteFile(path, []byte(req.Content), 0644)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "ok"})
	})

	api.POST("/files/create", func(c *gin.Context) {
		var req struct {
			Path  string `json:"path"`
			IsDir bool   `json:"is_dir"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		path := filepath.Clean(req.Path)

		if req.IsDir {
			err := os.MkdirAll(path, 0755)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		} else {
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			file.Close()
		}

		c.JSON(200, gin.H{"status": "ok"})
	})

	api.POST("/files/delete", func(c *gin.Context) {
		var req struct {
			Path string `json:"path"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		path := filepath.Clean(req.Path)

		// Safety checks
		if path == "/" || path == "/root" || path == "/etc" || path == "/bin" || path == "/usr" || path == "/var" {
			c.JSON(400, gin.H{"error": "Deleting critical system directories is blocked for safety"})
			return
		}

		err := os.RemoveAll(path)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "ok"})
	})

	api.POST("/files/rename", func(c *gin.Context) {
		var req struct {
			OldPath string `json:"old_path"`
			NewPath string `json:"new_path"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		oldPath := filepath.Clean(req.OldPath)
		newPath := filepath.Clean(req.NewPath)

		if oldPath == "/" || newPath == "/" {
			c.JSON(400, gin.H{"error": "Renaming root directory is blocked"})
			return
		}

		err := os.Rename(oldPath, newPath)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "ok"})
	})
}
