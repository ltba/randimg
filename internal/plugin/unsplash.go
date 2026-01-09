package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"randimg/internal/database"
	"randimg/internal/model"
)

// UnsplashPlugin Unsplash图源插件
type UnsplashPlugin struct {
	apiKey string
	client *http.Client
}

// NewUnsplashPlugin 创建Unsplash插件
func NewUnsplashPlugin(apiKey string) *UnsplashPlugin {
	return &UnsplashPlugin{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

// UnsplashPhoto Unsplash照片结构
type UnsplashPhoto struct {
	ID          string `json:"id"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Description string `json:"description"`
	URLs        struct {
		Raw     string `json:"raw"`
		Full    string `json:"full"`
		Regular string `json:"regular"`
	} `json:"urls"`
	User struct {
		Name string `json:"name"`
	} `json:"user"`
}

// FetchRandomPhotos 获取随机照片
func (p *UnsplashPlugin) FetchRandomPhotos(count int, query string) ([]UnsplashPhoto, error) {
	url := fmt.Sprintf("https://api.unsplash.com/photos/random?count=%d", count)
	if query != "" {
		url += fmt.Sprintf("&query=%s", query)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Client-ID "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unsplash API error: %d - %s", resp.StatusCode, string(body))
	}

	var photos []UnsplashPhoto
	if err := json.NewDecoder(resp.Body).Decode(&photos); err != nil {
		return nil, err
	}

	return photos, nil
}

// ImportPhotos 导入照片到数据库
func (p *UnsplashPlugin) ImportPhotos(categoryID uint, count int, query string) (int, error) {
	photos, err := p.FetchRandomPhotos(count, query)
	if err != nil {
		return 0, err
	}

	imported := 0
	for _, photo := range photos {
		// 检查是否已存在
		var existing model.Image
		if err := database.DB.Where("source_url = ?", photo.URLs.Regular).First(&existing).Error; err == nil {
			// 已存在，跳过
			continue
		}

		// 创建图片记录
		image := model.Image{
			SourceURL:  photo.URLs.Regular,
			Width:      &photo.Width,
			Height:     &photo.Height,
			Format:     "jpeg",
			Source:     fmt.Sprintf("Unsplash - %s", photo.User.Name),
			CategoryID: categoryID,
			Status:     "active",
		}

		if err := database.DB.Create(&image).Error; err != nil {
			// 记录错误但继续
			fmt.Printf("Failed to import photo %s: %v\n", photo.ID, err)
			continue
		}

		imported++
	}

	return imported, nil
}

// Example 使用示例
func Example() {
	// 创建插件实例
	plugin := NewUnsplashPlugin("YOUR_UNSPLASH_ACCESS_KEY")

	// 导入10张风景照片到分类ID为1的分类
	count, err := plugin.ImportPhotos(1, 10, "landscape")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Successfully imported %d photos\n", count)
}
