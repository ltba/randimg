// Package imageproxy 图片代理: 转发图源字节并可选转码.
// 模块定义见 .trellis/spec/arch/imageproxy.md.
package imageproxy

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif" // 注册解码器
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"time"

	"randimg/internal/store"

	"github.com/gin-gonic/gin"
)

// proxyCacheAge 代理响应缓存 2 天.
const proxyCacheAge = "public, max-age=172800"

// Handler 图片代理端点.
type Handler struct {
	client *http.Client
}

// NewHandler 构造代理 Handler; 图源超时集中此处.
func NewHandler() *Handler {
	return &Handler{client: &http.Client{Timeout: 30 * time.Second}}
}

// ProxyImage GET /api/proxy/:id?compress=false&format=jpeg
func (h *Handler) ProxyImage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	var img store.Image
	if err := store.DB.First(&img, id).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	compress := c.DefaultQuery("compress", "false") == "true" || c.Query("compress") == "1"
	targetFormat := c.Query("format")

	data, contentType, err := h.fetch(img.SourceURL)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch image"})
		return
	}

	if compress {
		data, contentType = h.reencode(data, contentType, targetFormat)
	}

	c.Header("Cache-Control", proxyCacheAge)
	c.Data(http.StatusOK, contentType, data)
}

// fetch 拉取图源字节与 Content-Type.
func (h *Handler) fetch(url string) ([]byte, string, error) {
	resp, err := h.client.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", errUnexpectedStatus(resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return data, resp.Header.Get("Content-Type"), nil
}

func errUnexpectedStatus(code int) error {
	return fmt.Errorf("source status %d", code)
}

// reencode 解码重编码: 仅 jpeg (质量 85) 与 png (无损);
// 其他格式或解码失败时原样返回原图字节与原 Content-Type.
func (h *Handler) reencode(data []byte, contentType, targetFormat string) ([]byte, string) {
	img, decoded, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, contentType
	}
	if targetFormat == "" {
		targetFormat = decoded
	}

	var buf bytes.Buffer
	switch targetFormat {
	case "jpeg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
			return data, contentType
		}
		return buf.Bytes(), "image/jpeg"
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return data, contentType
		}
		return buf.Bytes(), "image/png"
	default:
		return data, contentType
	}
}
