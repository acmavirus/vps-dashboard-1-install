package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

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

	// --- File Manager Pro Upgrades (v4.0) ---

	api.POST("/files/chmod", func(c *gin.Context) {
		var req struct {
			Path string `json:"path"`
			Mode string `json:"mode"` // e.g. "0755" or "0644"
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		path := filepath.Clean(req.Path)
		var mode uint32
		_, err := fmt.Sscanf(req.Mode, "%o", &mode)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid mode format"})
			return
		}
		err = os.Chmod(path, os.FileMode(mode))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	api.POST("/files/chown", func(c *gin.Context) {
		var req struct {
			Path  string `json:"path"`
			User  string `json:"user"`
			Group string `json:"group"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		path := filepath.Clean(req.Path)
		if runtime.GOOS == "windows" {
			c.JSON(200, gin.H{"status": "ok", "message": "Chown is not supported on Windows (Simulated success)"})
			return
		}
		cmd := exec.Command("chown", "-R", req.User+":"+req.Group, path)
		output, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error(), "details": string(output)})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	api.POST("/files/zip", func(c *gin.Context) {
		var req struct {
			Path    string `json:"path"`
			ZipPath string `json:"zip_path"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		path := filepath.Clean(req.Path)
		zipPath := filepath.Clean(req.ZipPath)

		err := zipDirectory(path, zipPath)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	api.POST("/files/unzip", func(c *gin.Context) {
		var req struct {
			Path     string `json:"path"`
			DestPath string `json:"dest_path"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		path := filepath.Clean(req.Path)
		destPath := filepath.Clean(req.DestPath)

		err := unzipExtract(path, destPath)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	api.GET("/files/download-folder", func(c *gin.Context) {
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
		if !info.IsDir() {
			c.JSON(400, gin.H{"error": "Path is not a directory"})
			return
		}

		tmpZip := filepath.Join(os.TempDir(), fmt.Sprintf("folder_download_%d.zip", time.Now().UnixNano()))
		err = zipDirectory(path, tmpZip)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to zip directory: " + err.Error()})
			return
		}

		c.File(tmpZip)
		go func() {
			time.Sleep(10 * time.Second)
			_ = os.Remove(tmpZip)
		}()
	})
}

func zipDirectory(source, target string) error {
	zipFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	info, err := os.Stat(source)
	if err != nil {
		return err
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		if baseDir != "" {
			relPath, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			if relPath == "." {
				return nil
			}
			header.Name = filepath.ToSlash(filepath.Join(baseDir, relPath))
		} else {
			header.Name = filepath.Base(path)
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}

func unzipExtract(source, target string) error {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	err = os.MkdirAll(target, 0755)
	if err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		
		cleanTarget := filepath.Clean(target)
		cleanPath := filepath.Clean(path)
		if !strings.HasPrefix(cleanPath, cleanTarget) {
			return fmt.Errorf("illegal file path in zip: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			_ = os.MkdirAll(path, 0755)
			continue
		}

		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			return err
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			fileReader.Close()
			return err
		}

		_, err = io.Copy(targetFile, fileReader)
		targetFile.Close()
		fileReader.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
