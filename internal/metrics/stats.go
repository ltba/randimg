package metrics

import (
	"net/http"
	"strconv"
	"time"

	"randimg/internal/store"

	"github.com/gin-gonic/gin"
)

// Handler 统计查询端点.
type Handler struct{}

// NewHandler 构造统计 Handler.
func NewHandler() *Handler { return &Handler{} }

// PublicStats GET /api/stats — 公开统计; 自身不计量.
func (h *Handler) PublicStats(c *gin.Context) {
	totalImages, err := store.CountActiveImages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load stats"})
		return
	}
	todayCalls, err := store.CountCallsSince(UTCDayStart())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load stats"})
		return
	}
	totalCalls, err := store.CountAllCalls()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load stats"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"total_images": totalImages,
		"today_calls":  todayCalls,
		"total_calls":  totalCalls,
	})
}

// AdminStats GET /api/admin/stats — 按 Channel 与时间区间查询; channel_id 必填, 日期口径 UTC.
func (h *Handler) AdminStats(c *gin.Context) {
	channelID, err := strconv.ParseUint(c.Query("channel_id"), 10, 64)
	if err != nil || channelID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_id is required"})
		return
	}

	var start, end *time.Time
	if s := c.Query("start_time"); s != "" {
		t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_time"})
			return
		}
		start = &t
	}
	if s := c.Query("end_time"); s != "" {
		t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_time"})
			return
		}
		end = &t
	}

	logs, total, err := store.ListCallLogs(uint(channelID), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load stats"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": total})
}

// OverviewStats GET /api/admin/stats/overview — 全局概览, 口径含匿名调用.
func (h *Handler) OverviewStats(c *gin.Context) {
	activeImages, err := store.CountActiveImages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load stats"})
		return
	}
	activeChannels, err := store.CountActiveChannels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load stats"})
		return
	}
	todayCalls, err := store.CountCallsSince(UTCDayStart())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load stats"})
		return
	}
	totalCalls, err := store.CountAllCalls()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load stats"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"active_images":   activeImages,
		"active_channels": activeChannels,
		"today_calls":     todayCalls,
		"total_calls":     totalCalls,
	})
}

// UTCDayStart 返回 UTC 今日零点.
func UTCDayStart() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
