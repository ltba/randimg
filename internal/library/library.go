// Package library 图库管理: 管理面图片与分类 CRUD, 批量导入与补全触发.
// 模块定义见 .trellis/spec/arch/library.md.
package library

import (
	"net/http"
	"strconv"

	"randimg/internal/metadata"
	"randimg/internal/store"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler 图库管理端点; 认证由接入网关的 AdminAuthMiddleware 提供.
type Handler struct {
	fetch *metadata.MetadataFetchService
}

// NewHandler 构造图库管理 Handler.
func NewHandler(fetch *metadata.MetadataFetchService) *Handler {
	return &Handler{fetch: fetch}
}

// ---------- 图片管理 ----------

type imageInput struct {
	SourceURL  string `json:"source_url" binding:"required"`
	Width      *int   `json:"width"`
	Height     *int   `json:"height"`
	Format     string `json:"format"`
	Source     string `json:"source"`
	CategoryID uint   `json:"category_id" binding:"required"`
	AutoFetch  bool   `json:"auto_fetch"`
}

// ListImages GET /api/admin/images — 管理面查询不受 active 限制.
func (h *Handler) ListImages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	q := store.DB.Model(&store.Image{}).Preload("Category")
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if c.Query("fetch_failed") == "true" {
		q = q.Where("fetch_fails >= ?", 3)
	}
	if category := c.Query("category"); category != "" {
		var cat store.Category
		if err := store.DB.Where("slug = ?", category).First(&cat).Error; err == nil {
			q = q.Where("category_id = ?", cat.ID)
		} else {
			c.JSON(http.StatusOK, gin.H{
				"data": []store.Image{},
				"pagination": gin.H{
					"page": page, "page_size": pageSize, "total": 0, "total_page": 0,
				},
			})
			return
		}
	}
	if search := c.Query("q"); search != "" {
		like := "%" + search + "%"
		q = q.Where("source_url LIKE ? OR source LIKE ?", like, like)
	}
	if category := c.Query("category"); category != "" {
		var cat store.Category
		if err := store.DB.Where("slug = ?", category).First(&cat).Error; err == nil {
			q = q.Where("category_id = ?", cat.ID)
		}
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list images"})
		return
	}
	var images []store.Image
	if err := q.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&images).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list images"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": images,
		"pagination": gin.H{
			"page": page, "page_size": pageSize, "total": total,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetImage GET /api/admin/images/:id
func (h *Handler) GetImage(c *gin.Context) {
	var img store.Image
	if err := store.DB.Preload("Category").First(&img, c.Param("id")).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}
	c.JSON(http.StatusOK, img)
}

// CreateImage POST /api/admin/images
func (h *Handler) CreateImage(c *gin.Context) {
	var input imageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	img := store.Image{
		SourceURL:  input.SourceURL,
		Width:      input.Width,
		Height:     input.Height,
		Format:     input.Format,
		Source:     input.Source,
		CategoryID: input.CategoryID,
		Status:     "active",
	}
	if err := store.DB.Create(&img).Error; err != nil {
		respondDBError(c, err, "source_url already exists")
		return
	}
	if input.AutoFetch {
		h.fetch.Enqueue(img.ID)
	}
	c.JSON(http.StatusCreated, img)
}

// BatchCreateImages POST /api/admin/images/batch — 单次最多 1000 条, 原子完成.
func (h *Handler) BatchCreateImages(c *gin.Context) {
	var input struct {
		Images []imageInput `json:"images" binding:"required,min=1,max=1000"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	images := make([]store.Image, len(input.Images))
	for i, item := range input.Images {
		images[i] = store.Image{
			SourceURL:  item.SourceURL,
			Width:      item.Width,
			Height:     item.Height,
			Format:     item.Format,
			Source:     item.Source,
			CategoryID: item.CategoryID,
			Status:     "active",
		}
	}

	if err := store.DB.Transaction(func(tx *gorm.DB) error {
		return tx.CreateInBatches(&images, 100).Error
	}); err != nil {
		respondDBError(c, err, "source_url already exists")
		return
	}

	pending := 0
	for i, item := range input.Images {
		if item.AutoFetch {
			if h.fetch.Enqueue(images[i].ID) {
				pending++
			}
		}
	}
	c.JSON(http.StatusCreated, gin.H{
		"success":       true,
		"count":         len(images),
		"images":        images,
		"fetch_pending": pending,
	})
}

// UpdateImage PUT /api/admin/images/:id — 可更新字段白名单见 spec F-006.
func (h *Handler) UpdateImage(c *gin.Context) {
	var img store.Image
	if err := store.DB.First(&img, c.Param("id")).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	var input struct {
		SourceURL  *string `json:"source_url"`
		Width      *int    `json:"width"`
		Height     *int    `json:"height"`
		Format     *string `json:"format"`
		CategoryID *uint   `json:"category_id"`
		Status     *string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
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
	if input.CategoryID != nil {
		updates["category_id"] = *input.CategoryID
	}
	if input.Status != nil {
		updates["status"] = *input.Status
	}

	if err := store.DB.Model(&img).Updates(updates).Error; err != nil {
		respondDBError(c, err, "source_url already exists")
		return
	}
	store.DB.First(&img, img.ID)
	c.JSON(http.StatusOK, img)
}

// DeleteImage DELETE /api/admin/images/:id
func (h *Handler) DeleteImage(c *gin.Context) {
	if err := store.DB.Unscoped().Delete(&store.Image{}, c.Param("id")).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Image deleted successfully"})
}

// BatchUpdateImages PUT /api/admin/images/batch — updates 只接受白名单字段.
func (h *Handler) BatchUpdateImages(c *gin.Context) {
	var input struct {
		ImageIDs []uint                 `json:"image_ids" binding:"required"`
		Updates  map[string]interface{} `json:"updates" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	allowed := map[string]bool{
		"source_url": true, "width": true, "height": true,
		"format": true, "category_id": true, "status": true,
	}
	filtered := map[string]interface{}{}
	for k, v := range input.Updates {
		if allowed[k] {
			filtered[k] = v
		}
	}
	if len(filtered) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "No updatable fields"})
		return
	}
	if err := store.DB.Model(&store.Image{}).Where("id IN ?", input.ImageIDs).Updates(filtered).Error; err != nil {
		respondDBError(c, err, "source_url already exists")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Batch update successful", "updated": len(input.ImageIDs)})
}

// BatchDeleteImages DELETE /api/admin/images/batch
func (h *Handler) BatchDeleteImages(c *gin.Context) {
	var input struct {
		ImageIDs []uint `json:"image_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := store.DB.Unscoped().Where("id IN ?", input.ImageIDs).Delete(&store.Image{}).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete images"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Batch delete successful", "deleted": len(input.ImageIDs)})
}

// AutoFetchInfo POST /api/admin/images/auto-fetch — 补全触发面, 入队异步执行.
func (h *Handler) AutoFetchInfo(c *gin.Context) {
	var input struct {
		ImageIDs []uint `json:"image_ids"`
		All      bool   `json:"all"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var images []store.Image
	switch {
	case input.All:
		store.DB.Where("width IS NULL OR height IS NULL OR format = '' OR format IS NULL").Find(&images)
	case len(input.ImageIDs) > 0:
		store.DB.Where("id IN ?", input.ImageIDs).Find(&images)
	default:
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Please specify image_ids or set all=true"})
		return
	}

	queued := 0
	for _, img := range images {
		if h.fetch.Enqueue(img.ID) {
			queued++
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Fetch tasks enqueued", "queued": queued})
}

// ---------- 分类管理 ----------

// ListCategories GET /api/admin/categories
func (h *Handler) ListCategories(c *gin.Context) {
	var categories []store.Category
	if err := store.DB.Find(&categories).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to list categories"})
		return
	}
	c.JSON(http.StatusOK, categories)
}

// CreateCategory POST /api/admin/categories
func (h *Handler) CreateCategory(c *gin.Context) {
	var input struct {
		Name        string `json:"name" binding:"required"`
		Slug        string `json:"slug" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cat := store.Category{Name: input.Name, Slug: input.Slug, Description: input.Description}
	if err := store.DB.Create(&cat).Error; err != nil {
		respondDBError(c, err, "Category already exists")
		return
	}
	c.JSON(http.StatusCreated, cat)
}

// UpdateCategory PUT /api/admin/categories/:id
func (h *Handler) UpdateCategory(c *gin.Context) {
	var cat store.Category
	if err := store.DB.First(&cat, c.Param("id")).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}
	var input struct {
		Name        *string `json:"name"`
		Slug        *string `json:"slug"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.Slug != nil {
		updates["slug"] = *input.Slug
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if err := store.DB.Model(&cat).Updates(updates).Error; err != nil {
		respondDBError(c, err, "Category already exists")
		return
	}
	store.DB.First(&cat, cat.ID)
	c.JSON(http.StatusOK, cat)
}

// DeleteCategory DELETE /api/admin/categories/:id — 非空禁删.
func (h *Handler) DeleteCategory(c *gin.Context) {
	var count int64
	store.DB.Model(&store.Image{}).Where("category_id = ?", c.Param("id")).Count(&count)
	if count > 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Cannot delete category with images"})
		return
	}
	if err := store.DB.Delete(&store.Category{}, c.Param("id")).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
}

// respondDBError 唯一冲突转 400, 其余 500.
func respondDBError(c *gin.Context, err error, conflictMsg string) {
	if store.IsUniqueViolation(err) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": conflictMsg})
		return
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
}
