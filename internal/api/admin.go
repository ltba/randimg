package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"randimg/internal/database"
	"randimg/internal/model"
	"randimg/internal/service"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AdminAPI 管理API处理器
type AdminAPI struct {
	statService *service.StatService
}

// NewAdminAPI 创建管理API处理器
func NewAdminAPI() *AdminAPI {
	return &AdminAPI{
		statService: service.GetStatService(),
	}
}

// ========== 图片管理 ==========

// ListImages 获取图片列表
// GET /api/admin/images?page=1&page_size=20&category=acg&status=active
func (api *AdminAPI) ListImages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	category := c.Query("category")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := database.DB.Model(&model.Image{}).Preload("Category")

	if category != "" {
		var cat model.Category
		if err := database.DB.Where("slug = ?", category).First(&cat).Error; err == nil {
			query = query.Where("category_id = ?", cat.ID)
		}
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var images []model.Image
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&images).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": images,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetImage 获取单个图片
// GET /api/admin/images/:id
func (api *AdminAPI) GetImage(c *gin.Context) {
	id := c.Param("id")

	var image model.Image
	if err := database.DB.Preload("Category").First(&image, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	c.JSON(http.StatusOK, image)
}

// BatchCreateImages 批量创建图片
// POST /api/admin/images/batch
func (api *AdminAPI) BatchCreateImages(c *gin.Context) {
	var input struct {
		Images []struct {
			SourceURL  string `json:"source_url" binding:"required"`
			Width      *int   `json:"width"`
			Height     *int   `json:"height"`
			Format     string `json:"format"`
			Source     string `json:"source"`
			CategoryID uint   `json:"category_id" binding:"required"`
			AutoFetch  bool   `json:"auto_fetch"`
		} `json:"images" binding:"required,min=1,max=1000"` // 最多1000条
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 构建图片数据（先插入数据库）
	images := make([]model.Image, len(input.Images))
	needFetchIDs := make([]uint, 0)

	for i, item := range input.Images {
		images[i] = model.Image{
			SourceURL:  item.SourceURL,
			Width:      item.Width,
			Height:     item.Height,
			Format:     item.Format,
			Source:     item.Source,
			CategoryID: item.CategoryID,
			Status:     "active",
		}
	}

	// 使��事务批量插入
	tx := database.DB.Begin()
	if err := tx.CreateInBatches(images, 100).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	tx.Commit()

	// 收集需要auto_fetch的图片ID
	for i, item := range input.Images {
		if item.AutoFetch && (item.Width == nil || item.Height == nil || item.Format == "") {
			needFetchIDs = append(needFetchIDs, images[i].ID)
		}
	}

	// 异步提交fetch任务
	if len(needFetchIDs) > 0 {
		go func() {
			fetchService := service.GetImageFetchService()
			for _, id := range needFetchIDs {
				fetchService.AddTask(id)
			}
		}()
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":       true,
		"count":         len(images),
		"images":        images,
		"fetch_pending": len(needFetchIDs),
	})
}

// CreateImage 创建图片
// POST /api/admin/images
func (api *AdminAPI) CreateImage(c *gin.Context) {
	var input struct {
		SourceURL  string `json:"source_url" binding:"required"`
		Width      *int   `json:"width"`
		Height     *int   `json:"height"`
		Format     string `json:"format"`
		Source     string `json:"source"`
		CategoryID uint   `json:"category_id" binding:"required"`
		AutoFetch  bool   `json:"auto_fetch"` // 是否自动获取图片信息
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 先插入数据库
	image := model.Image{
		SourceURL:  input.SourceURL,
		Width:      input.Width,
		Height:     input.Height,
		Format:     input.Format,
		Source:     input.Source,
		CategoryID: input.CategoryID,
		Status:     "active",
	}

	if err := database.DB.Create(&image).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 如果需要auto_fetch，异步提交任务
	if input.AutoFetch && (input.Width == nil || input.Height == nil || input.Format == "") {
		go func() {
			fetchService := service.GetImageFetchService()
			fetchService.AddTask(image.ID)
		}()
	}

	c.JSON(http.StatusCreated, image)
}

// UpdateImage 更新图片
// PUT /api/admin/images/:id
func (api *AdminAPI) UpdateImage(c *gin.Context) {
	id := c.Param("id")

	var image model.Image
	if err := database.DB.First(&image, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	var input struct {
		SourceURL  *string `json:"source_url"`
		Width      *int    `json:"width"`
		Height     *int    `json:"height"`
		Format     *string `json:"format"`
		Source     *string `json:"source"`
		CategoryID *uint   `json:"category_id"`
		Status     *string `json:"status"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if input.SourceURL != nil {
		updates["source_url"] = *input.SourceURL
	}
	if input.Width != nil {
		updates["width"] = *input.Width
	}
	if input.Height != nil {
		updates["height"] = *input.Height
	}
	if input.Format != nil {
		updates["format"] = *input.Format
	}
	if input.Source != nil {
		updates["source"] = *input.Source
	}
	if input.CategoryID != nil {
		updates["category_id"] = *input.CategoryID
	}
	if input.Status != nil {
		updates["status"] = *input.Status
	}

	if err := database.DB.Model(&image).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, image)
}

// DeleteImage 删除图片（硬删除）
// DELETE /api/admin/images/:id
func (api *AdminAPI) DeleteImage(c *gin.Context) {
	id := c.Param("id")

	var image model.Image
	if err := database.DB.First(&image, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// 硬删除
	if err := database.DB.Unscoped().Delete(&image).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Image deleted successfully"})
}

// AutoFetchImageInfo 自动获取图片信息
// POST /api/admin/images/auto-fetch
func (api *AdminAPI) AutoFetchImageInfo(c *gin.Context) {
	var input struct {
		ImageIDs []uint `json:"image_ids"` // 可选，指定图片ID列表
		All      bool   `json:"all"`       // 是否处理所有缺失信息的图片
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var images []model.Image

	if input.All {
		// 查询所有缺失信息的图片
		database.DB.Where("width IS NULL OR height IS NULL OR format = '' OR format IS NULL").Find(&images)
	} else if len(input.ImageIDs) > 0 {
		// 查询指定的图片
		database.DB.Where("id IN ?", input.ImageIDs).Find(&images)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please specify image_ids or set all=true"})
		return
	}

	if len(images) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No images to process",
			"updated": 0,
			"failed":  0,
		})
		return
	}

	infoService := service.NewImageInfoService()
	updated := 0
	failed := 0
	errors := make([]string, 0)

	for _, image := range images {
		info, err := infoService.GetImageInfo(image.SourceURL)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("Image %d: %s", image.ID, err.Error()))
			continue
		}

		// 更新图片信息
		updates := make(map[string]interface{})
		if image.Width == nil && info.Width > 0 {
			updates["width"] = info.Width
		}
		if image.Height == nil && info.Height > 0 {
			updates["height"] = info.Height
		}
		if (image.Format == "" || image.Format == "null") && info.Format != "" {
			updates["format"] = info.Format
		}

		if len(updates) > 0 {
			if err := database.DB.Model(&image).Updates(updates).Error; err != nil {
				failed++
				errors = append(errors, fmt.Sprintf("Image %d: failed to update - %s", image.ID, err.Error()))
			} else {
				updated++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Processed %d images", len(images)),
		"updated": updated,
		"failed":  failed,
		"errors":  errors,
	})
}

// BatchUpdateImages 批量更新图片
// PUT /api/admin/images/batch
func (api *AdminAPI) BatchUpdateImages(c *gin.Context) {
	var input struct {
		ImageIDs []uint                 `json:"image_ids" binding:"required"`
		Updates  map[string]interface{} `json:"updates" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(input.ImageIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image_ids cannot be empty"})
		return
	}

	// 批量更新
	result := database.DB.Model(&model.Image{}).Where("id IN ?", input.ImageIDs).Updates(input.Updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Batch update successful",
		"updated": result.RowsAffected,
	})
}

// BatchDeleteImages 批量删除图片
// DELETE /api/admin/images/batch
func (api *AdminAPI) BatchDeleteImages(c *gin.Context) {
	var input struct {
		ImageIDs []uint `json:"image_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(input.ImageIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image_ids cannot be empty"})
		return
	}

	// 批量硬删除
	result := database.DB.Unscoped().Where("id IN ?", input.ImageIDs).Delete(&model.Image{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Batch delete successful",
		"deleted": result.RowsAffected,
	})
}

// ========== 分类管理 ==========

// ListCategories 获取分类列表
// GET /api/admin/categories
func (api *AdminAPI) ListCategories(c *gin.Context) {
	var categories []model.Category
	if err := database.DB.Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// CreateCategory 创建分类
// POST /api/admin/categories
func (api *AdminAPI) CreateCategory(c *gin.Context) {
	var input struct {
		Name        string `json:"name" binding:"required"`
		Slug        string `json:"slug" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category := model.Category{
		Name:        input.Name,
		Slug:        input.Slug,
		Description: input.Description,
	}

	if err := database.DB.Create(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, category)
}

// UpdateCategory 更新分类
// PUT /api/admin/categories/:id
func (api *AdminAPI) UpdateCategory(c *gin.Context) {
	id := c.Param("id")

	var category model.Category
	if err := database.DB.First(&category, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	var input struct {
		Name        *string `json:"name"`
		Slug        *string `json:"slug"`
		Description *string `json:"description"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.Slug != nil {
		updates["slug"] = *input.Slug
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}

	if err := database.DB.Model(&category).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, category)
}

// DeleteCategory 删除分类
// DELETE /api/admin/categories/:id
func (api *AdminAPI) DeleteCategory(c *gin.Context) {
	id := c.Param("id")

	// 检查是否有图片使用该分类
	var count int64
	database.DB.Model(&model.Image{}).Where("category_id = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete category with images"})
		return
	}

	if err := database.DB.Delete(&model.Category{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
}

// ========== API Key管理 ==========

// ListAPIKeys 获取API Key列表
// GET /api/admin/api-keys
func (api *AdminAPI) ListAPIKeys(c *gin.Context) {
	var keys []model.APIKey
	if err := database.DB.Order("created_at DESC").Find(&keys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, keys)
}

// CreateAPIKey 创建API Key
// POST /api/admin/api-keys
func (api *AdminAPI) CreateAPIKey(c *gin.Context) {
	var input struct {
		Key       string `json:"key"`        // 可选，自定义key
		RateLimit int    `json:"rate_limit" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Trim空格并处理key
	key := strings.TrimSpace(input.Key)
	if key == "" {
		key = generateAPIKey()
	}

	// 检查key是否已存在
	var existing model.APIKey
	if err := database.DB.Where("key = ?", key).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API key already exists"})
		return
	}

	apiKey := model.APIKey{
		Key:       key,
		RateLimit: input.RateLimit,
		Status:    "active",
	}

	if err := database.DB.Create(&apiKey).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, apiKey)
}

// UpdateAPIKey 更新API Key
// PUT /api/admin/api-keys/:id
func (api *AdminAPI) UpdateAPIKey(c *gin.Context) {
	id := c.Param("id")

	var apiKey model.APIKey
	if err := database.DB.First(&apiKey, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	var input struct {
		Key       *string `json:"key"`
		RateLimit *int    `json:"rate_limit"`
		Status    *string `json:"status"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})

	// 如果要更新key，trim空格并检查是否已存在
	if input.Key != nil {
		trimmedKey := strings.TrimSpace(*input.Key)
		if trimmedKey != apiKey.Key {
			var existing model.APIKey
			if err := database.DB.Where("key = ?", trimmedKey).First(&existing).Error; err == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "API key already exists"})
				return
			}
			updates["key"] = trimmedKey
		}
	}

	if input.RateLimit != nil {
		updates["rate_limit"] = *input.RateLimit
	}
	if input.Status != nil {
		updates["status"] = *input.Status
	}

	if err := database.DB.Model(&apiKey).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新加载更新后的数据
	database.DB.First(&apiKey, id)

	c.JSON(http.StatusOK, apiKey)
}

// DeleteAPIKey 删除API Key
// DELETE /api/admin/api-keys/:id
func (api *AdminAPI) DeleteAPIKey(c *gin.Context) {
	id := c.Param("id")

	if err := database.DB.Delete(&model.APIKey{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted successfully"})
}

// ========== 统计查询 ==========

// GetStats 获取统计数据
// GET /api/admin/stats?api_key_id=1&start_time=2024-01-01&end_time=2024-01-31
func (api *AdminAPI) GetStats(c *gin.Context) {
	apiKeyIDStr := c.Query("api_key_id")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	if apiKeyIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_key_id is required"})
		return
	}

	apiKeyID, err := strconv.ParseUint(apiKeyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid api_key_id"})
		return
	}

	var startTime, endTime time.Time
	if startTimeStr != "" {
		startTime, _ = time.Parse("2006-01-02", startTimeStr)
	}
	if endTimeStr != "" {
		endTime, _ = time.Parse("2006-01-02", endTimeStr)
	}

	logs, err := api.statService.GetStats(uint(apiKeyID), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	count, err := api.statService.GetStatsCount(uint(apiKeyID), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"total": count,
	})
}

// GetStatsOverview 获取统计概览
// GET /api/admin/stats/overview
func (api *AdminAPI) GetStatsOverview(c *gin.Context) {
	// 总图片数
	var totalImages int64
	database.DB.Model(&model.Image{}).Where("status = ?", "active").Count(&totalImages)

	// 总API Key数
	var totalAPIKeys int64
	database.DB.Model(&model.APIKey{}).Where("status = ?", "active").Count(&totalAPIKeys)

	// 今日调用次数
	today := time.Now().Truncate(24 * time.Hour)
	var todayCalls int64
	database.DB.Model(&model.APIUsageLog{}).Where("requested_at >= ?", today).Count(&todayCalls)

	// 总调用次数
	var totalCalls int64
	database.DB.Model(&model.APIUsageLog{}).Count(&totalCalls)

	c.JSON(http.StatusOK, gin.H{
		"total_images":    totalImages,
		"total_api_keys":  totalAPIKeys,
		"today_calls":     todayCalls,
		"total_calls":     totalCalls,
	})
}

// generateAPIKey 生成随机API key
func generateAPIKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
