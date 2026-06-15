package main

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
)

var authMiddleware = func(c *gin.Context) {
	// Check token from Header or Query parameters (for SSE)
	token := c.GetHeader("Authorization")
	if token == "" {
		token = c.Query("token")
	}

	if token != authToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}
	c.Next()
}

func registerAuthRoutes(r *gin.Engine) {
	// Unprotected routes
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
			twoFAEnabled := getSetting("2fa_enabled", "false") == "true"
			if twoFAEnabled {
				c.JSON(200, gin.H{
					"status": "require_2fa",
				})
				return
			}

			c.JSON(200, gin.H{
				"status": "ok",
				"token":  authToken,
			})
		} else {
			c.JSON(401, gin.H{"error": "Sai tài khoản hoặc mật khẩu"})
		}
	})

	r.POST("/api/login/verify-2fa", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Code     string `json:"code"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		if req.Username == adminUser && req.Password == adminPass {
			twoFAEnabled := getSetting("2fa_enabled", "false") == "true"
			if !twoFAEnabled {
				c.JSON(200, gin.H{
					"status": "ok",
					"token":  authToken,
				})
				return
			}

			secret := getSetting("2fa_secret", "")
			valid := totp.Validate(req.Code, secret)
			if valid {
				c.JSON(200, gin.H{
					"status": "ok",
					"token":  authToken,
				})
			} else {
				c.JSON(401, gin.H{"error": "Mã xác thực 2FA không chính xác"})
			}
		} else {
			c.JSON(401, gin.H{"error": "Sai tài khoản hoặc mật khẩu"})
		}
	})
}

func registerProtectedAuthRoutes(api *gin.RouterGroup) {
	// Protected 2FA Settings endpoints
	api.GET("/settings/2fa/status", func(c *gin.Context) {
		twoFAEnabled := getSetting("2fa_enabled", "false") == "true"
		c.JSON(200, gin.H{
			"enabled": twoFAEnabled,
		})
	})

	api.POST("/settings/2fa/generate", func(c *gin.Context) {
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "AcmaDash",
			AccountName: adminUser,
		})
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to generate TOTP key"})
			return
		}

		var buf bytes.Buffer
		img, err := key.Image(200, 200)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to generate QR image"})
			return
		}
		
		err = png.Encode(&buf, img)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to encode QR image"})
			return
		}

		qrBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())
		qrCodeDataURI := "data:image/png;base64," + qrBase64

		c.JSON(200, gin.H{
			"secret":  key.Secret(),
			"qr_code": qrCodeDataURI,
		})
	})

	api.POST("/settings/2fa/enable", func(c *gin.Context) {
		var req struct {
			Secret string `json:"secret"`
			Code   string `json:"code"`
		}
		if err := c.BindJSON(&req); err != nil || req.Secret == "" || req.Code == "" {
			c.JSON(400, gin.H{"error": "Secret and Code are required"})
			return
		}

		valid := totp.Validate(req.Code, req.Secret)
		if !valid {
			c.JSON(400, gin.H{"error": "Mã xác thực không chính xác. Vui lòng kiểm tra lại."})
			return
		}

		_ = saveSetting("2fa_secret", req.Secret)
		_ = saveSetting("2fa_enabled", "true")

		c.JSON(200, gin.H{"status": "ok"})
	})

	api.POST("/settings/2fa/disable", func(c *gin.Context) {
		var req struct {
			Code string `json:"code"`
		}
		if err := c.BindJSON(&req); err != nil || req.Code == "" {
			c.JSON(400, gin.H{"error": "Code is required"})
			return
		}

		secret := getSetting("2fa_secret", "")
		valid := totp.Validate(req.Code, secret)
		if !valid {
			c.JSON(400, gin.H{"error": "Mã xác thực không chính xác. Không thể tắt 2FA."})
			return
		}

		_ = saveSetting("2fa_enabled", "false")
		_ = saveSetting("2fa_secret", "")

		c.JSON(200, gin.H{"status": "ok"})
	})
}
