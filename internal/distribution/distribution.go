// Package distribution 图片分发: 随机选取与公开查询.
// 模块定义见 .trellis/spec/arch/distribution.md.
package distribution

import (
	"fmt"
	"net/http"
	"strconv"

	"randimg/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/mssola/user_agent"
)

// Handler 公开查询端点.
type Handler struct{}

// NewHandler 构造分发 Handler.
func NewHandler() *Handler { return &Handler{} }

// deviceFromRequest 设备类型判定: URL 参数 > User-Agent 解析 > 默认 pc.
func deviceFromRequest(c *gin.Context) string {
	if device := c.Query("device"); device != "" {
		return device
	}
	uaString := c.GetHeader("User-Agent")
	if uaString == "" {
		return "pc"
	}
	if user_agent.New(uaString).Mobile() {
		return "mobile"
	}
	return "pc"
}

// RandomImage GET /api/random?category=&device=&output=redirect|proxy|json&compress=
func (h *Handler) RandomImage(c *gin.Context) {
	// 随机端点不被缓存.
	c.Header("Cache-Control", "no-store")

	category := c.Query("category")
	device := deviceFromRequest(c)
	output := c.DefaultQuery("output", "redirect")
	compress := c.DefaultQuery("compress", "false") == "true" || c.Query("compress") == "1"

	q := store.DB.Model(&store.Image{}).Where("status = ?", "active")
	if category != "" {
		var cat store.Category
		if err := store.DB.Where("slug = ?", category).First(&cat).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}
		q = q.Where("category_id = ?", cat.ID)
	}

	// 宽高过滤: pc 宽 >= 高, mobile 高 >= 宽, 其他值不过滤.
	switch device {
	case "pc":
		q = q.Where("width IS NOT NULL AND height IS NOT NULL AND width >= height")
	case "mobile":
		q = q.Where("width IS NOT NULL AND height IS NOT NULL AND height >= width")
	}

	var img store.Image
	if err := q.Preload("Category").Order("RANDOM()").First(&img).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "No images found"})
		return
	}

	switch output {
	case "redirect":
		c.Redirect(http.StatusFound, img.SourceURL)
	case "proxy":
		proxyURL := fmt.Sprintf("/api/proxy/%d", img.ID)
		if compress {
			proxyURL += "?compress=true"
		}
		c.Redirect(http.StatusFound, proxyURL)
	case "json":
		c.JSON(http.StatusOK, gin.H{
			"id":       img.ID,
			"url":      img.SourceURL,
			"proxy":    fmt.Sprintf("/api/proxy/%d", img.ID),
			"width":    img.Width,
			"height":   img.Height,
			"format":   img.Format,
			"category": img.Category,
		})
	default:
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid output parameter"})
	}
}

// ListImages GET /api/images?page=1&page_size=20&category=&device=
func (h *Handler) ListImages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	category := c.Query("category")
	device := c.Query("device")

	if page < 1 {
		page = 1
	}
	// 归一: 小于 1 取默认 20, 大于 100 收敛到 100.
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	q := store.DB.Model(&store.Image{}).Preload("Category").Where("status = ?", "active")
	if category != "" {
		var cat store.Category
		if err := store.DB.Where("slug = ?", category).First(&cat).Error; err == nil {
			q = q.Where("category_id = ?", cat.ID)
		} else {
			// slug 不存在: 过滤无命中, 返回空集.
			c.JSON(http.StatusOK, gin.H{
				"data": []gin.H{},
				"pagination": gin.H{
					"page": page, "page_size": pageSize, "total": 0, "total_page": 0,
				},
			})
			return
		}
	}
	if device == "pc" {
		q = q.Where("width IS NOT NULL AND height IS NOT NULL AND width >= height")
	} else if device == "mobile" {
		q = q.Where("width IS NOT NULL AND height IS NOT NULL AND height >= width")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list images"})
		return
	}
	var images []store.Image
	if err := q.Offset((page - 1) * pageSize).Limit(pageSize).Find(&images).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list images"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": publicImages(images),
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// publicImages 公开视图: 剔除 source 字段 (公开面不暴露来源信息).
func publicImages(images []store.Image) []gin.H {
	out := make([]gin.H, len(images))
	for i := range images {
		out[i] = gin.H{
			"id":          images[i].ID,
			"source_url":  images[i].SourceURL,
			"width":       images[i].Width,
			"height":      images[i].Height,
			"format":      images[i].Format,
			"status":      images[i].Status,
			"category_id": images[i].CategoryID,
			"category":    images[i].Category,
			"created_at":  images[i].CreatedAt,
		}
	}
	return out
}

// ListCategories GET /api/categories — 全量, 无分页.
func (h *Handler) ListCategories(c *gin.Context) {
	var categories []store.Category
	if err := store.DB.Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list categories"})
		return
	}
	c.JSON(http.StatusOK, categories)
}
