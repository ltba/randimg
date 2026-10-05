// Package channel Channel 管理: 接入渠道的创建, 停用与配额调整.
// 模块定义见 .trellis/spec/arch/channel.md.
package channel

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"

	"randimg/internal/store"

	"github.com/gin-gonic/gin"
)

// channelIDPattern 自定义 channel_id 约束: 8-64 字符, 字母数字连字符下划线.
var channelIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{8,64}$`)

// Handler Channel 管理端点; 认证由接入网关的 AdminAuthMiddleware 提供.
type Handler struct{}

// NewHandler 构造 Channel 管理 Handler.
func NewHandler() *Handler { return &Handler{} }

// ListChannels GET /api/admin/channels — 创建时间倒序, 含 last_used_at.
func (h *Handler) ListChannels(c *gin.Context) {
	chs, err := store.ListChannels()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to list channels"})
		return
	}
	views := make([]gin.H, 0, len(chs))
	for i := range chs {
		views = append(views, channelView(&chs[i]))
	}
	c.JSON(http.StatusOK, gin.H{"data": views})
}

// CreateChannel POST /api/admin/channels
func (h *Handler) CreateChannel(c *gin.Context) {
	var input struct {
		ChannelID      string   `json:"channel_id"`
		RateLimit      int      `json:"rate_limit" binding:"required,min=1"`
		AllowedOrigins []string `json:"allowed_origins"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := input.ChannelID
	if id == "" {
		id = generateChannelID()
	} else if !channelIDPattern.MatchString(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "channel_id must be 8-64 chars of letters, digits, hyphen or underscore",
		})
		return
	}
	if _, err := store.GetChannelByChannelID(id); err == nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "channel_id already exists"})
		return
	}

	ch := store.Channel{
		ChannelID:      id,
		RateLimit:      input.RateLimit,
		Status:         "active",
		AllowedOrigins: marshalOrigins(input.AllowedOrigins),
	}
	if err := store.CreateChannel(&ch); err != nil {
		if store.IsUniqueViolation(err) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "channel_id already exists"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create channel"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":              ch.ID,
		"channel_id":      ch.ChannelID,
		"rate_limit":      ch.RateLimit,
		"status":          ch.Status,
		"allowed_origins": input.AllowedOrigins,
		"created_at":      ch.CreatedAt,
		"last_used_at":    ch.LastUsedAt,
	})
}

// UpdateChannel PUT /api/admin/channels/:id — 可修改 rate_limit, status, allowed_origins.
func (h *Handler) UpdateChannel(c *gin.Context) {
	ch, err := store.GetChannelByID(parseID(c))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	var input struct {
		RateLimit      *int      `json:"rate_limit"`
		Status         *string   `json:"status"`
		AllowedOrigins *[]string `json:"allowed_origins"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if input.RateLimit != nil {
		if *input.RateLimit < 1 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "rate_limit must be >= 1"})
			return
		}
		updates["rate_limit"] = *input.RateLimit
	}
	if input.Status != nil {
		if *input.Status != "active" && *input.Status != "disabled" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "status must be active or disabled"})
			return
		}
		updates["status"] = *input.Status
	}
	if input.AllowedOrigins != nil {
		updates["allowed_origins"] = marshalOrigins(*input.AllowedOrigins)
	}

	if err := store.DB.Model(ch).Updates(updates).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel"})
		return
	}
	updated, _ := store.GetChannelByID(ch.ID)
	c.JSON(http.StatusOK, channelView(updated))
}

// DeleteChannel DELETE /api/admin/channels/:id
func (h *Handler) DeleteChannel(c *gin.Context) {
	if err := store.DeleteChannel(parseID(c)); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete channel"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Channel deleted successfully"})
}

// channelView 输出视图: allowed_origins 还原为字符串数组.
func channelView(ch *store.Channel) gin.H {
	var origins []string
	json.Unmarshal([]byte(normalizeOriginsJSON(ch.AllowedOrigins)), &origins)
	if origins == nil {
		origins = []string{}
	}
	return gin.H{
		"id":              ch.ID,
		"channel_id":      ch.ChannelID,
		"rate_limit":      ch.RateLimit,
		"status":          ch.Status,
		"allowed_origins": origins,
		"created_at":      ch.CreatedAt,
		"last_used_at":    ch.LastUsedAt,
	}
}

func generateChannelID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func marshalOrigins(origins []string) string {
	if len(origins) == 0 {
		return ""
	}
	data, _ := json.Marshal(origins)
	return string(data)
}

func normalizeOriginsJSON(raw string) string {
	if raw == "" {
		return "[]"
	}
	return raw
}

func parseID(c *gin.Context) uint {
	var id uint
	json.Unmarshal([]byte(c.Param("id")), &id)
	return id
}
