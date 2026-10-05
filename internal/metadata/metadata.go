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

	"gorm.io/gorm"
)

const (
	queueCap      = 10000
	scanLimit     = 1000
	maxFetchFails = 3   // 连续失败达此次数后不再重扫
	batchFlush    = 50  // 结果批量落库条数阈值
	flushInterval = 500 * time.Millisecond // 结果批量落库时间窗
	rangeLimit    = 256 * 1024 // 只取文件头部字节, 足够解析尺寸与格式
)

// MetadataFetchService 后台元数据补全服务.
type MetadataFetchService struct {
	taskQueue chan uint
	stopChan  chan struct{}
	wg        sync.WaitGroup
	writers   sync.WaitGroup
	results   chan fetchResult
	workers   int
	client    *http.Client
}

// fetchResult 单图抓取结果; 由 writer 聚合批量落库.
type fetchResult struct {
	imageID   uint
	updates   map[string]interface{} // 空缺字段补全
	failed    bool                     // 抓取失败, fetch_fails +1
	clearFail bool                     // 成功且此前有失败计数, 清零
}

// NewMetadataFetchService 构造服务; workers 与超时集中此处配置.
func NewMetadataFetchService(workers int) *MetadataFetchService {
	if workers < 1 {
		workers = 1
	}
	return &MetadataFetchService{
		taskQueue: make(chan uint, queueCap),
		stopChan:  make(chan struct{}),
		results:   make(chan fetchResult, queueCap),
		workers:   workers,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Start 启动 worker 与结果 writer, 并扫描缺失元数据的 active 图片 (单轮上限 1000).
func (s *MetadataFetchService) Start() {
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker()
	}
	s.writers.Add(1)
	go s.writer()
	log.Printf("[metadata] service started with %d workers", s.workers)
	s.scanPending()
}

// Stop 停止 worker 与 writer; writer 先排空结果队列再退出.
func (s *MetadataFetchService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
	close(s.results)
	s.writers.Wait()
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

// scanPending 启动扫描: 缺失宽/高/文件格式的 active 图片; 连续失败 >= maxFetchFails 的跳过.
func (s *MetadataFetchService) scanPending() {
	var images []store.Image
	if err := store.DB.
		Where("(width IS NULL OR height IS NULL OR format = '' OR format IS NULL) AND status = ? AND fetch_fails < ?", "active", maxFetchFails).
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
		s.results <- fetchResult{imageID: imageID, failed: true}
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
	select {
	case s.results <- fetchResult{imageID: imageID, updates: updates, clearFail: img.FetchFails > 0}:
	default:
		// 结果队列满时丢弃, 留待下轮扫描.
		log.Printf("[metadata] result queue full, dropping result for image %d", imageID)
	}
}

// writer 聚合抓取结果批量落库: 按批 (batchFlush 条) 或时间窗 (flushInterval) 合并事务,
// 避免 SQLite 单写者下逐条事务的 fsync 排队.
func (s *MetadataFetchService) writer() {
	defer s.writers.Done()
	batch := make([]fetchResult, 0, batchFlush)
	for {
		select {
		case r, ok := <-s.results:
			if !ok {
				s.flush(batch)
				return
			}
			batch = append(batch, r)
			if len(batch) >= batchFlush {
				batch = s.flush(batch)
			}
		case <-time.After(flushInterval):
			if len(batch) > 0 {
				batch = s.flush(batch)
			}
		}
	}
}

func (s *MetadataFetchService) flush(batch []fetchResult) []fetchResult {
	if len(batch) == 0 {
		return batch[:0]
	}
	err := store.DB.Transaction(func(tx *gorm.DB) error {
		for _, r := range batch {
			if r.failed {
				if err := tx.Model(&store.Image{}).Where("id = ?", r.imageID).
					UpdateColumn("fetch_fails", gorm.Expr("fetch_fails + 1")).Error; err != nil {
					return err
				}
				continue
			}
			if len(r.updates) > 0 {
				if err := tx.Model(&store.Image{}).Where("id = ?", r.imageID).Updates(r.updates).Error; err != nil {
					return err
				}
			}
			if r.clearFail {
				if err := tx.Model(&store.Image{}).Where("id = ?", r.imageID).
					UpdateColumn("fetch_fails", 0).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("[metadata] batch flush failed (%d results): %v", len(batch), err)
	}
	return batch[:0]
}

// imageInfo 图源元数据.
type imageInfo struct {
	Width  int
	Height int
	Format string
}

// fetchInfo 抓取图源头部, 只读解码配置获取尺寸与格式.
// jpeg SOF / png IHDR / gif 头部均在文件前部, 用 Range 请求只取前 rangeLimit 字节;
// 服务器忽略 Range 时返回 200 全量, 行为退化为整图流式读取.
func (s *MetadataFetchService) fetchInfo(url string) (*imageInfo, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", rangeLimit-1))
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch image: %w", err)
	}
	defer resp.Body.Close()
	// 206 = 服务器响应了 Range; 200 = 忽略 Range 回全量.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
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
