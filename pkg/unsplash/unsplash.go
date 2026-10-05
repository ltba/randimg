// Package unsplash Unsplash 图源导入: 库级交付, 无 HTTP 路由, 由宿主程序实例化调用.
// 模块定义见 .trellis/spec/arch/unsplash.md.
package unsplash

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"randimg/internal/store"
)

// UnsplashPlugin Unsplash 图源插件.
type UnsplashPlugin struct {
	accessKey string
	client    *http.Client
}

// NewUnsplashPlugin 构造插件.
func NewUnsplashPlugin(accessKey string) *UnsplashPlugin {
	return &UnsplashPlugin{
		accessKey: accessKey,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// unsplashPhoto Unsplash 照片结构.
type unsplashPhoto struct {
	ID     string `json:"id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URLs   struct {
		Regular string `json:"regular"`
	} `json:"urls"`
	User struct {
		Name string `json:"name"`
	} `json:"user"`
}

// fetchRandomPhotos 按 query 检索随机照片.
func (p *UnsplashPlugin) fetchRandomPhotos(count int, query string) ([]unsplashPhoto, error) {
	url := fmt.Sprintf("https://api.unsplash.com/photos/random?count=%d", count)
	if query != "" {
		url += fmt.Sprintf("&query=%s", query)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Client-ID "+p.accessKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unsplash API error: %d - %s", resp.StatusCode, string(body))
	}

	var photos []unsplashPhoto
	if err := json.NewDecoder(resp.Body).Decode(&photos); err != nil {
		return nil, err
	}
	return photos, nil
}

// ImportPhotos 导入照片入库: 宽高取 Unsplash 返回, format 记 jpeg,
// 来源记 Unsplash 加作者名; source_url 已存在时跳过.
func (p *UnsplashPlugin) ImportPhotos(categoryID uint, count int, query string) (int, error) {
	photos, err := p.fetchRandomPhotos(count, query)
	if err != nil {
		return 0, err
	}

	imported := 0
	for _, photo := range photos {
		var existing store.Image
		if err := store.DB.Where("source_url = ?", photo.URLs.Regular).First(&existing).Error; err == nil {
			continue
		}
		width, height := photo.Width, photo.Height
		image := store.Image{
			SourceURL:  photo.URLs.Regular,
			Width:      &width,
			Height:     &height,
			Format:     "jpeg",
			Source:     fmt.Sprintf("Unsplash - %s", photo.User.Name),
			CategoryID: categoryID,
			Status:     "active",
		}
		if err := store.DB.Create(&image).Error; err != nil {
			continue
		}
		imported++
	}
	return imported, nil
}
