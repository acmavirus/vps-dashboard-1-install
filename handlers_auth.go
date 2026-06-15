package main

import (
	"github.com/gin-gonic/gin"
)

var authMiddleware = func(c *gin.Context) {
	// Kiểm tra token từ Header hoặc Query (cho SSE)
	token := c.GetHeader("Authorization")
	if token == "" {
		token = c.Query("token")
	}

	if token != authToken {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}
	c.Next()
}

func registerAuthRoutes(r *gin.Engine) {
	r.POST("/api/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		if req.Username == adminUser && req.Password == adminPass {
			c.JSON(200, gin.H{
				"status": "ok",
				"token":  authToken,
			})
		} else {
			c.JSON(401, gin.H{"error": "Sai tài khoản hoặc mật khẩu"})
		}
	})
}
