package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin: func(r *http.Request) bool {
		return true // Access control is validated via token
	},
}

type wsResizeMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func registerTerminalRoutes(api *gin.RouterGroup) {
	// WebSocket SSH Terminal route
	api.GET("/terminal/ws", func(c *gin.Context) {
		token := c.Query("token")
		if token != authToken {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Terminal WS upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		// Determine shell executable
		var shell string
		var args []string
		if runtime.GOOS == "windows" {
			shell = "powershell.exe"
			args = []string{"-NoLogo"}
		} else {
			shell = "bash"
			args = []string{"-i"} // Interactive shell
		}

		cmd := exec.Command(shell, args...)
		cmd.Env = os.Environ()
		// Set basic TERM env var so terminal apps like htop work correctly
		cmd.Env = append(cmd.Env, "TERM=xterm-256color")

		// Start command with a PTY
		ptyFile, err := pty.Start(cmd)
		if err != nil {
			log.Printf("Failed to start PTY shell: %v", err)
			_ = conn.WriteMessage(websocket.TextMessage, []byte("\r\n[Error starting shell PTY]\r\n"))
			return
		}
		defer ptyFile.Close()

		// Go routine: PTY -> WebSocket
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := ptyFile.Read(buf)
				if err != nil {
					// PTY closed
					_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "PTY exited"))
					return
				}
				if n > 0 {
					err = conn.WriteMessage(websocket.BinaryMessage, buf[:n])
					if err != nil {
						return
					}
				}
			}
		}()

		// Main loop: WebSocket -> PTY
		for {
			messageType, payload, err := conn.ReadMessage()
			if err != nil {
				break
			}

			if messageType == websocket.TextMessage {
				// Check if it's a resize event JSON
				var resize wsResizeMessage
				if err := json.Unmarshal(payload, &resize); err == nil && resize.Type == "resize" {
					_ = pty.Setsize(ptyFile, &pty.Winsize{
						Rows: resize.Rows,
						Cols: resize.Cols,
					})
					continue
				}

				// Otherwise, write keyboard text input to PTY
				_, _ = ptyFile.Write(payload)
			} else if messageType == websocket.BinaryMessage {
				_, _ = ptyFile.Write(payload)
			}
		}

		// Clean up process on exit
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
}
