package service

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strings"
)

// ImageProxyService 图片代理服务
type ImageProxyService struct {
	client *http.Client
}

// NewImageProxyService 创建图片代理服务
func NewImageProxyService() *ImageProxyService {
	return &ImageProxyService{
		client: &http.Client{
			Timeout: 30 * 1e9, // 30秒超时
		},
	}
}

// ProxyImage 代理图片
func (s *ImageProxyService) ProxyImage(sourceURL string, compress bool, format string) ([]byte, string, error) {
	// 获取原图
	resp, err := s.client.Get(sourceURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to fetch image: status %d", resp.StatusCode)
	}

	// 读取图片数据
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image: %w", err)
	}

	// 获取原始Content-Type
	contentType := resp.Header.Get("Content-Type")

	// 如果不需要压缩，直接返回
	if !compress {
		return data, contentType, nil
	}

	// 需要压缩，进行格式转换
	return s.compressImage(data, format)
}

// compressImage 压缩图片
func (s *ImageProxyService) compressImage(data []byte, targetFormat string) ([]byte, string, error) {
	// 解码图片
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	// 如果目标格式为空，使用原格式
	if targetFormat == "" {
		targetFormat = format
	}

	// 转换为目标格式
	var buf bytes.Buffer
	var contentType string

	switch strings.ToLower(targetFormat) {
	case "jpeg", "jpg":
		// 转换为JPEG
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
			return nil, "", fmt.Errorf("failed to encode jpeg: %w", err)
		}
		contentType = "image/jpeg"

	case "png":
		// 转换为PNG
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", fmt.Errorf("failed to encode png: %w", err)
		}
		contentType = "image/png"

	default:
		// 不支持的格式，返回原图
		return data, "application/octet-stream", nil
	}

	return buf.Bytes(), contentType, nil
}
