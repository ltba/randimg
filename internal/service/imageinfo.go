package service

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"
)

// ImageInfoService 图片信息服务
type ImageInfoService struct {
	client *http.Client
}

// NewImageInfoService 创建图片信息服务
func NewImageInfoService() *ImageInfoService {
	return &ImageInfoService{
		client: &http.Client{
			Timeout: 30 * 1e9, // 30秒超时
		},
	}
}

// ImageInfo 图片信息
type ImageInfo struct {
	Width  int
	Height int
	Format string
}

// GetImageInfo 获取图片信息
func (s *ImageInfoService) GetImageInfo(url string) (*ImageInfo, error) {
	// 获取图片
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch image: status %d", resp.StatusCode)
	}

	// 解码图片获取尺寸和格式（只读取配置，不读取整个图片）
	img, format, err := image.DecodeConfig(resp.Body)
	if err != nil {
		// 如果解码失败，尝试从Content-Type获取格式
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg") {
			format = "jpeg"
		} else if strings.Contains(contentType, "png") {
			format = "png"
		} else if strings.Contains(contentType, "gif") {
			format = "gif"
		} else if strings.Contains(contentType, "webp") {
			format = "webp"
		}

		// 如果无法获取格式，返回错误
		if format == "" {
			return nil, fmt.Errorf("failed to decode image config: %w", err)
		}

		// 无法获取尺寸，返回部分信息
		return &ImageInfo{
			Width:  0,
			Height: 0,
			Format: format,
		}, nil
	}

	return &ImageInfo{
		Width:  img.Width,
		Height: img.Height,
		Format: format,
	}, nil
}
