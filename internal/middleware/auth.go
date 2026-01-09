package middleware

import (
	"net/http"
	"os"
	"randimg/internal/database"
	"randimg/internal/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware API Key认证中间件（可选认证）
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从header或query获取API key
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		// 如果没有API key，允许通过但不设置context
		// 这样同源访问不需要API key，跨域访问需要API key（通过CORS中间件控制）
		if apiKey == "" {
			c.Next()
			return
		}

		// 验证API key
		var key model.APIKey
		if err := database.DB.Where("key = ? AND status = ?", apiKey, "active").First(&key).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}

		// 更新最后使用时间
		now := time.Now()
		database.DB.Model(&key).Update("last_used_at", now)

		// 将API key信息存入context
		c.Set("api_key", &key)
		c.Next()
	}
}

// AdminAuthMiddleware 管理员认证中间件
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从header获取admin token
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			c.Abort()
			return
		}

		// 移除 "Bearer " 前缀
		token = strings.TrimPrefix(token, "Bearer ")

		// 从环境变量读取admin token
		adminToken := os.Getenv("ADMIN_TOKEN")
		if adminToken == "" {
			adminToken = "admin_secret_token" // 默认值
		}

		if token != adminToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid admin token"})
			c.Abort()
			return
		}

		c.Next()
	}
}
