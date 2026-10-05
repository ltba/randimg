// Package access 接入网关: 公开面 CORS, Channel 校验与计量, 限流 (Channel 窗口与匿名桶), 管理面认证.
// 模块定义见 .trellis/spec/arch/access.md.
package access

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"randimg/internal/metrics"
	"randimg/internal/store"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware 公开面 CORS: /api 且非 /api/admin 的响应全局放行, 预检统一 204 且携带 CORS 头.
// 必须全局挂载: 预检 OPTIONS 无匹配路由, 不进路由组中间件链.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if !strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/api/admin") {
			c.Next()
			return
		}
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// AccessMiddleware Channel 校验与计量: channel_id 仅取 query 参数;
// 不存在 404, 非 active 403; 来源绑定严格模式; 通过后非阻塞提交计量.
func AccessMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Query("channel_id")
		if id == "" {
			metrics.Record(metrics.CallEntry{ChannelID: nil, Path: c.FullPath()})
			c.Next()
			return
		}

		ch, err := store.GetChannelByChannelID(id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		if ch.Status != "active" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Channel disabled"})
			return
		}
		if !originAllowed(c, ch.AllowedOrigins) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Origin not allowed"})
			return
		}

		chID := ch.ID
		metrics.Record(metrics.CallEntry{ChannelID: &chID, Path: c.FullPath()})
		c.Set("channel", ch)
		c.Next()
	}
}

// originAllowed 来源绑定: 配置白名单后 Origin/Referer 源必须命中; 无来源同样拒绝.
func originAllowed(c *gin.Context, allowedJSON string) bool {
	raw := strings.TrimSpace(allowedJSON)
	if raw == "" || raw == "[]" {
		return true
	}
	var allowed []string
	if err := json.Unmarshal([]byte(raw), &allowed); err != nil {
		log.Printf("[access] invalid allowed_origins config, rejecting: %v", err)
		return false
	}

	origin := c.GetHeader("Origin")
	if origin == "" {
		if ref := c.GetHeader("Referer"); ref != "" {
			if u, err := url.Parse(ref); err == nil && u.Scheme != "" && u.Host != "" {
				origin = u.Scheme + "://" + u.Host
			}
		}
	}
	if origin == "" {
		return false
	}
	for _, a := range allowed {
		if strings.EqualFold(strings.TrimRight(a, "/"), strings.TrimRight(origin, "/")) {
			return true
		}
	}
	return false
}

// AdminAuthMiddleware 管理面认证: Bearer ADMIN_TOKEN, 失败 401.
// 未配置时回退默认值; 启动告警由组合根输出.
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			return
		}
		if token != adminToken() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid admin token"})
			return
		}
		c.Next()
	}
}

func adminToken() string {
	if t := os.Getenv("ADMIN_TOKEN"); t != "" {
		return t
	}
	return "admin_secret_token"
}

// RateLimitMiddleware Channel 窗口 + 匿名桶; anonLimit 为匿名桶阈值 (60 秒窗口).
func RateLimitMiddleware(anonLimit int) gin.HandlerFunc {
	anon := newWindow(anonLimit)
	return func(c *gin.Context) {
		var w *window
		limit := anonLimit

		if v, ok := c.Get("channel"); ok && v != nil {
			ch, ok := v.(*store.Channel)
			if !ok {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Invalid channel context"})
				return
			}
			w = registry().get("channel:"+strconv.FormatUint(uint64(ch.ID), 10), ch.RateLimit)
			limit = ch.RateLimit
		} else {
			w = anon
		}

		if !w.allow() {
			c.Header("Retry-After", "60")
			c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": 60,
			})
			return
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(w.remaining()))
		c.Next()
	}
}
