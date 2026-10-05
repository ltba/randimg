// Package metadata 元数据补全: 异步抓取图源, 只填空缺的宽高与文件格式.
// 模块定义见 .trellis/spec/arch/metadata.md.
package metadata

import (
	"fmt"
	"image"
	// 注册解码器, 供 DecodeConfig 探测格式.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"randimg/internal/store"
)

const (
	queueCap  = 10000
	scanLimit = 1000
)

// MetadataFetchService 后台元数据补全服务.
type MetadataFetchService struct {
	taskQueue chan uint
	stopChan  chan struct{}
	wg        sync.WaitGroup
	workers   int
	client    *http.Client
}

// NewMetadataFetchService 构造服务; workers 与超时集中此处配置.
func NewMetadataFetchService(workers int) *MetadataFetchService {
	if workers < 1 {
		workers = 1
	}
	return &MetadataFetchService{
		taskQueue: make(chan uint, queueCap),
		stopChan:  make(chan struct{}),
		workers:   workers,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Start 启动 worker 并扫描缺失元数据的 active 图片 (单轮上限 1000).
func (s *MetadataFetchService) Start() {
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker()
	}
	log.Printf("[metadata] service started with %d workers", s.workers)
	s.scanPending()
}

// Stop 停止全部 worker.
func (s *MetadataFetchService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
	log.Println("[metadata] service stopped")
}

// Enqueue 入队补全任务; 队列满时丢弃并记日志, 返回 false.
func (s *MetadataFetchService) Enqueue(imageID uint) bool {
	select {
	case s.taskQueue <- imageID:
		return true
	default:
		log.Printf("[metadata] queue full, dropping task for image %d", imageID)
		return false
	}
}

// scanPending 启动扫描: 缺失宽/高/文件格式的 active 图片.
func (s *MetadataFetchService) scanPending() {
	var images []store.Image
	if err := store.DB.
		Where("(width IS NULL OR height IS NULL OR format = '' OR format IS NULL) AND status = ?", "active").
		Limit(scanLimit).Find(&images).Error; err != nil {
		log.Printf("[metadata] scan failed: %v", err)
		return
	}
	for _, img := range images {
		s.Enqueue(img.ID)
	}
	if len(images) > 0 {
		log.Printf("[metadata] scanned %d images with missing info", len(images))
	}
}

func (s *MetadataFetchService) worker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.stopChan:
			return
		case imageID := <-s.taskQueue:
			s.process(imageID)
		}
	}
}

func (s *MetadataFetchService) process(imageID uint) {
	var img store.Image
	if err := store.DB.First(&img, imageID).Error; err != nil {
		return
	}
	// 信息已完整的跳过.
	if img.Width != nil && img.Height != nil && img.Format != "" {
		return
	}

	info, err := s.fetchInfo(img.SourceURL)
	if err != nil {
		log.Printf("[metadata] fetch failed for image %d: %v", imageID, err)
		return
	}

	// 只填空缺字段, 不覆盖已有值.
	updates := make(map[string]interface{})
	if img.Width == nil && info.Width > 0 {
		updates["width"] = info.Width
	}
	if img.Height == nil && info.Height > 0 {
		updates["height"] = info.Height
	}
	if img.Format == "" && info.Format != "" {
		updates["format"] = info.Format
	}
	if len(updates) > 0 {
		if err := store.DB.Model(&img).Updates(updates).Error; err != nil {
			log.Printf("[metadata] update failed for image %d: %v", imageID, err)
		}
	}
}

// imageInfo 图源元数据.
type imageInfo struct {
	Width  int
	Height int
	Format string
}

// fetchInfo 抓取图源, 只读解码配置获取尺寸与格式.
func (s *MetadataFetchService) fetchInfo(url string) (*imageInfo, error) {
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch image: status %d", resp.StatusCode)
	}

	cfg, format, err := image.DecodeConfig(resp.Body)
	if err != nil {
		// 解码失败时从 Content-Type 尽力提取格式.
		ct := resp.Header.Get("Content-Type")
		for _, mapping := range [][2]string{
			{"jpeg", "jpeg"}, {"jpg", "jpeg"}, {"png", "png"}, {"gif", "gif"}, {"webp", "webp"},
		} {
			if strings.Contains(ct, mapping[0]) {
				return &imageInfo{Format: mapping[1]}, nil
			}
		}
		return nil, fmt.Errorf("decode image config: %w", err)
	}
	return &imageInfo{Width: cfg.Width, Height: cfg.Height, Format: format}, nil
}
