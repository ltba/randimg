package service

import (
	"log"
	"randimg/internal/database"
	"randimg/internal/model"
	"sync"
	"time"
)

// StatService 统计服务
type StatService struct {
	buffer    []model.APIUsageLog
	mu        sync.Mutex
	batchSize int
	flushTime time.Duration
	stopCh    chan struct{}
}

var statServiceInstance *StatService
var statServiceOnce sync.Once

// GetStatService 获取统计服务单例
func GetStatService() *StatService {
	statServiceOnce.Do(func() {
		statServiceInstance = &StatService{
			buffer:    make([]model.APIUsageLog, 0, 100),
			batchSize: 50,
			flushTime: 10 * time.Second,
			stopCh:    make(chan struct{}),
		}
		statServiceInstance.start()
	})
	return statServiceInstance
}

// start 启动后台刷新任务
func (s *StatService) start() {
	go func() {
		ticker := time.NewTicker(s.flushTime)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.flush()
			case <-s.stopCh:
				s.flush() // 最后刷新一次
				return
			}
		}
	}()
}

// Stop 停止统计服务
func (s *StatService) Stop() {
	close(s.stopCh)
}

// Record 记录API调用
func (s *StatService) Record(log model.APIUsageLog) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buffer = append(s.buffer, log)

	// 如果达到批量大小，立即刷新
	if len(s.buffer) >= s.batchSize {
		go s.flush()
	}
}

// flush 批量写入数据库
func (s *StatService) flush() {
	s.mu.Lock()
	if len(s.buffer) == 0 {
		s.mu.Unlock()
		return
	}

	// 复制buffer并清空
	logs := make([]model.APIUsageLog, len(s.buffer))
	copy(logs, s.buffer)
	s.buffer = s.buffer[:0]
	s.mu.Unlock()

	// 批量插入
	if err := database.DB.CreateInBatches(logs, 100).Error; err != nil {
		log.Printf("Failed to flush stats: %v", err)
		// TODO: 可以考虑重试机制或写入日志文件
	} else {
		log.Printf("Flushed %d stat records to database", len(logs))
	}
}

// GetStats 获取统计数据
func (s *StatService) GetStats(apiKeyID uint, startTime, endTime time.Time) ([]model.APIUsageLog, error) {
	var logs []model.APIUsageLog
	query := database.DB.Where("api_key_id = ?", apiKeyID)

	if !startTime.IsZero() {
		query = query.Where("requested_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("requested_at <= ?", endTime)
	}

	err := query.Order("requested_at DESC").Limit(1000).Find(&logs).Error
	return logs, err
}

// GetStatsCount 获取统计计数
func (s *StatService) GetStatsCount(apiKeyID uint, startTime, endTime time.Time) (int64, error) {
	var count int64
	query := database.DB.Model(&model.APIUsageLog{}).Where("api_key_id = ?", apiKeyID)

	if !startTime.IsZero() {
		query = query.Where("requested_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("requested_at <= ?", endTime)
	}

	err := query.Count(&count).Error
	return count, err
}
