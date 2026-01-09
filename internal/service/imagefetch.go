package service

import (
	"log"
	"randimg/internal/database"
	"randimg/internal/model"
	"sync"
)

// ImageFetchService 图片信息异步获取服务
type ImageFetchService struct {
	taskQueue   chan uint
	workers     int
	stopChan    chan struct{}
	wg          sync.WaitGroup
	infoService *ImageInfoService
}

var (
	fetchServiceInstance *ImageFetchService
	fetchServiceOnce     sync.Once
)

// GetImageFetchService 获取图片fetch服务单例
func GetImageFetchService() *ImageFetchService {
	fetchServiceOnce.Do(func() {
		fetchServiceInstance = NewImageFetchService(10) // 10个worker
		fetchServiceInstance.Start()
	})
	return fetchServiceInstance
}

// NewImageFetchService 创建图片fetch服务
func NewImageFetchService(workers int) *ImageFetchService {
	return &ImageFetchService{
		taskQueue:   make(chan uint, 10000), // 队列容量10000
		workers:     workers,
		stopChan:    make(chan struct{}),
		infoService: NewImageInfoService(),
	}
}

// Start 启动worker
func (s *ImageFetchService) Start() {
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
	log.Printf("ImageFetchService started with %d workers", s.workers)

	// 启动时扫描未完成的任务
	go s.scanPendingTasks()
}

// scanPendingTasks 扫描数据库中缺失信息的图片
func (s *ImageFetchService) scanPendingTasks() {
	log.Println("Scanning for pending fetch tasks...")

	var images []model.Image
	// 查找缺失信息的图片
	if err := database.DB.Where("(width IS NULL OR height IS NULL OR format = '' OR format IS NULL) AND status = ?", "active").
		Limit(1000).Find(&images).Error; err != nil {
		log.Printf("Failed to scan pending tasks: %v", err)
		return
	}

	if len(images) > 0 {
		log.Printf("Found %d images with missing info, adding to fetch queue", len(images))
		for _, image := range images {
			s.AddTask(image.ID)
		}
	} else {
		log.Println("No pending fetch tasks found")
	}
}

// Stop 停止服务
func (s *ImageFetchService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
	log.Println("ImageFetchService stopped")
}

// AddTask 添加fetch任务
func (s *ImageFetchService) AddTask(imageID uint) {
	select {
	case s.taskQueue <- imageID:
		// 任务已加入队列
	default:
		log.Printf("Task queue full, dropping task for image %d", imageID)
	}
}

// worker 处理任务
func (s *ImageFetchService) worker(id int) {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopChan:
			return
		case imageID := <-s.taskQueue:
			s.processTask(imageID)
		}
	}
}

// processTask 处理单个任务
func (s *ImageFetchService) processTask(imageID uint) {
	// 查询图片
	var image model.Image
	if err := database.DB.First(&image, imageID).Error; err != nil {
		log.Printf("Worker: failed to find image %d: %v", imageID, err)
		return
	}

	// 如果已有完整信息，跳过
	if image.Width != nil && image.Height != nil && image.Format != "" {
		return
	}

	// 获取图片信息
	info, err := s.infoService.GetImageInfo(image.SourceURL)
	if err != nil {
		log.Printf("Worker: failed to fetch info for image %d: %v", imageID, err)
		return
	}

	// 更新数据库
	updates := make(map[string]interface{})
	if image.Width == nil && info.Width > 0 {
		updates["width"] = info.Width
	}
	if image.Height == nil && info.Height > 0 {
		updates["height"] = info.Height
	}
	if image.Format == "" && info.Format != "" {
		updates["format"] = info.Format
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&image).Updates(updates).Error; err != nil {
			log.Printf("Worker: failed to update image %d: %v", imageID, err)
			return
		}
		log.Printf("Worker: updated image %d with %v", imageID, updates)
	}
}

// GetQueueSize 获取队列大小
func (s *ImageFetchService) GetQueueSize() int {
	return len(s.taskQueue)
}
