package api

import (
	"fmt"
	"net/http"
	"randimg/internal/database"
	"randimg/internal/model"
	"randimg/internal/service"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mssola/user_agent"
)

// PublicAPI 公开API处理器
type PublicAPI struct {
	proxyService *service.ImageProxyService
	statService  *service.StatService
}

// NewPublicAPI 创建公开API处理器
func NewPublicAPI() *PublicAPI {
	return &PublicAPI{
		proxyService: service.NewImageProxyService(),
		statService:  service.GetStatService(),
	}
}

// getDeviceFromRequest 从请求中智能识别设备类型
// 优先级：URL参数 > User-Agent解析 > 默认值(pc)
func getDeviceFromRequest(c *gin.Context) string {
	// 1. 如果URL明确指定了device参数，直接使用
	if device := c.Query("device"); device != "" {
		return device
	}

	// 2. 从User-Agent解析设备类型
	uaString := c.GetHeader("User-Agent")
	if uaString == "" {
		return "pc" // 无UA时默认pc
	}

	ua := user_agent.New(uaString)

	// 移动设备（包括手机和平板）
	if ua.Mobile() {
		return "mobile"
	}

	// 3. 默认返回pc
	return "pc"
}

// ListImages 获取图片列表（公开API）
// GET /api/images?page=1&page_size=20&category=acg&device=pc
func (api *PublicAPI) ListImages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	category := c.Query("category")
	device := c.Query("device")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := database.DB.Model(&model.Image{}).Preload("Category").Where("status = ?", "active")

	if category != "" {
		var cat model.Category
		if err := database.DB.Where("slug = ?", category).First(&cat).Error; err == nil {
			query = query.Where("category_id = ?", cat.ID)
		}
	}

	if device == "pc" {
		query = query.Where("width IS NOT NULL AND height IS NOT NULL AND width > height")
	} else if device == "mobile" {
		query = query.Where("width IS NOT NULL AND height IS NOT NULL AND height > width")
	}

	var total int64
	query.Count(&total)

	var images []model.Image
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&images).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPage := int((total + int64(pageSize) - 1) / int64(pageSize))

	c.JSON(http.StatusOK, gin.H{
		"data": images,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": totalPage,
		},
	})
}

// ListCategories 获取分类列表（公开API）
// GET /api/categories
func (api *PublicAPI) ListCategories(c *gin.Context) {
	var categories []model.Category
	if err := database.DB.Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// RandomImage 随机图片接口
// GET /api/random?category=acg&device=pc&format=redirect|proxy|json&compress=false
func (api *PublicAPI) RandomImage(c *gin.Context) {
	// 获取参数
	category := c.Query("category")
	device := getDeviceFromRequest(c) // 智能识别设备类型
	format := c.DefaultQuery("format", "redirect")
	compressStr := c.DefaultQuery("compress", "false")
	compress := compressStr == "true" || compressStr == "1"

	// 构建查询
	query := database.DB.Where("status = ?", "active")

	// 如果指定了分类
	if category != "" {
		var cat model.Category
		if err := database.DB.Where("slug = ?", category).First(&cat).Error; err == nil {
			query = query.Where("category_id = ?", cat.ID)
		}
	}

	// 根据device参数筛选图片（基于宽高比）
	if device == "pc" {
		// PC端：横屏图片（宽>高）
		query = query.Where("width IS NOT NULL AND height IS NOT NULL AND width > height")
	} else if device == "mobile" {
		// 移动端：竖屏图片（高>宽）
		query = query.Where("width IS NOT NULL AND height IS NOT NULL AND height > width")
	}
	// 如果device为其他值，返回所有图片（包括正方形）

	// 随机获取一张图片
	var image model.Image
	if err := query.Preload("Category").Order("RANDOM()").First(&image).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No images found"})
		api.recordStat(c)
		return
	}

	// 记录统计
	api.recordStat(c)

	// 根据format返回不同格式
	switch format {
	case "redirect":
		// 302重定向到原图（不缓存，保证每次随机）
		c.Redirect(http.StatusFound, image.SourceURL)

	case "proxy":
		// 代理模式：302重定向到proxy接口（让Cloudflare缓存固定URL）
		proxyURL := fmt.Sprintf("/api/proxy/%d", image.ID)
		if compress {
			proxyURL += "?compress=true"
		}
		c.Redirect(http.StatusFound, proxyURL)

	case "json":
		// JSON格式（不缓存，保证每次随机）
		c.JSON(http.StatusOK, gin.H{
			"id":       image.ID,
			"url":      image.SourceURL,
			"proxy":    fmt.Sprintf("/api/proxy/%d", image.ID),
			"width":    image.Width,
			"height":   image.Height,
			"format":   image.Format,
			"source":   image.Source,
			"category": image.Category,
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format parameter"})
	}
}

// ProxyImage 图片代理接口
// GET /api/proxy/:id?compress=false&format=webp
func (api *PublicAPI) ProxyImage(c *gin.Context) {
	// 获取图片ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	// 获取参数
	compressStr := c.DefaultQuery("compress", "false")
	compress := compressStr == "true" || compressStr == "1"
	targetFormat := c.Query("format")

	// 查询图片
	var image model.Image
	if err := database.DB.Preload("Category").First(&image, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		api.recordStat(c)
		return
	}

	// 记录统计
	api.recordStat(c)

	// 代理图片
	data, contentType, err := api.proxyService.ProxyImage(image.SourceURL, compress, targetFormat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Cache-Control", "public, max-age=172800") // 2天缓存
	c.Header("Content-Type", contentType)
	c.Data(http.StatusOK, contentType, data)
}

// GetPublicStats 获取公开统计数据（无需认证）
// GET /api/stats
func (api *PublicAPI) GetPublicStats(c *gin.Context) {
	// 总图片数
	var totalImages int64
	database.DB.Model(&model.Image{}).Where("status = ?", "active").Count(&totalImages)

	// 今日调用次数
	today := time.Now().Truncate(24 * time.Hour)
	var todayCalls int64
	database.DB.Model(&model.APIUsageLog{}).Where("requested_at >= ?", today).Count(&todayCalls)

	// 总调用次数
	var totalCalls int64
	database.DB.Model(&model.APIUsageLog{}).Count(&totalCalls)

	c.JSON(http.StatusOK, gin.H{
		"total_images": totalImages,
		"today_calls":  todayCalls,
		"total_calls":  totalCalls,
	})
}

// recordStat 记录统计信息 , imageID *uint, statusCode int
func (api *PublicAPI) recordStat(c *gin.Context) {
	apiKeyInterface, exists := c.Get("api_key")
	if !exists {
		return
	}

	apiKey, ok := apiKeyInterface.(*model.APIKey)
	if !ok {
		return
	}

	log := model.APIUsageLog{
		APIKeyID:    apiKey.ID,
		RequestedAt: time.Now(),
	}

	api.statService.Record(log)
}
