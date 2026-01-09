package middleware

import (
	"fmt"
	"net/http"
	"randimg/internal/model"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter 限流器
type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*slidingWindow
}

// slidingWindow 滑动窗口
type slidingWindow struct {
	requests  []time.Time
	limit     int
	window    time.Duration
	mu        sync.Mutex
}

var globalLimiter *RateLimiter

func init() {
	globalLimiter = &RateLimiter{
		limiters: make(map[string]*slidingWindow),
	}

	// 定期清理过期的限流器
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			globalLimiter.cleanup()
		}
	}()
}

// newSlidingWindow 创建滑动窗口
func newSlidingWindow(limit int, window time.Duration) *slidingWindow {
	return &slidingWindow{
		requests: make([]time.Time, 0),
		limit:    limit,
		window:   window,
	}
}

// allow 检查是否允许请求
func (sw *slidingWindow) allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.window)

	// 移除过期的请求记录
	validRequests := make([]time.Time, 0)
	for _, t := range sw.requests {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}
	sw.requests = validRequests

	// 检查是否超过限制
	if len(sw.requests) >= sw.limit {
		return false
	}

	// 记录本次请求
	sw.requests = append(sw.requests, now)
	return true
}

// remaining 返回剩余请求次数
func (sw *slidingWindow) remaining() int {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.window)

	count := 0
	for _, t := range sw.requests {
		if t.After(cutoff) {
			count++
		}
	}

	remaining := sw.limit - count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// getLimiter 获取或创建限流器
func (rl *RateLimiter) getLimiter(key string, limit int) *slidingWindow {
	rl.mu.RLock()
	limiter, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if exists {
		// 如果限制值改变了，更新限制
		limiter.mu.Lock()
		limiter.limit = limit
		limiter.mu.Unlock()
		return limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 双重检查
	if limiter, exists := rl.limiters[key]; exists {
		return limiter
	}

	limiter = newSlidingWindow(limit, time.Minute)
	rl.limiters[key] = limiter
	return limiter
}

// cleanup 清理长时间未使用的限流器
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, limiter := range rl.limiters {
		limiter.mu.Lock()
		if len(limiter.requests) == 0 || now.Sub(limiter.requests[len(limiter.requests)-1]) > 10*time.Minute {
			delete(rl.limiters, key)
		}
		limiter.mu.Unlock()
	}
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从context获取API key信息
		apiKeyInterface, exists := c.Get("api_key")
		if !exists {
			// 没有API key，跳过限流（同源访问）
			c.Next()
			return
		}

		apiKey, ok := apiKeyInterface.(*model.APIKey)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid API key type"})
			c.Abort()
			return
		}

		// 获取限流器
		limiter := globalLimiter.getLimiter(apiKey.Key, apiKey.RateLimit)

		// 检查是否允许请求
		if !limiter.allow() {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", apiKey.RateLimit))
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		// 设置响应头
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", apiKey.RateLimit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", limiter.remaining()))

		c.Next()
	}
}
