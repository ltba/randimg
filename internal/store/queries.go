package store

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// ErrNotFound 通用未命中.
var ErrNotFound = gorm.ErrRecordNotFound

// GetChannelByChannelID 按公开标识查询 Channel.
func GetChannelByChannelID(channelID string) (*Channel, error) {
	var ch Channel
	if err := DB.Where("channel_id = ?", channelID).First(&ch).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

// ListChannels 按 ID 倒序列出全部 Channel.
func ListChannels() ([]Channel, error) {
	var chs []Channel
	err := DB.Order("id DESC").Find(&chs).Error
	return chs, err
}

// CreateChannel / UpdateChannel / DeleteChannel 由 library 层校验后调用.
func CreateChannel(ch *Channel) error { return DB.Create(ch).Error }
func UpdateChannel(ch *Channel) error { return DB.Save(ch).Error }
func DeleteChannel(id uint) error     { return DB.Delete(&Channel{}, id).Error }
func GetChannelByID(id uint) (*Channel, error) {
	var ch Channel
	if err := DB.First(&ch, id).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

// CountActiveImages active 图片数.
func CountActiveImages() (int64, error) {
	var n int64
	err := DB.Model(&Image{}).Where("status = ?", "active").Count(&n).Error
	return n, err
}

// CountActiveChannels active Channel 数.
func CountActiveChannels() (int64, error) {
	var n int64
	err := DB.Model(&Channel{}).Where("status = ?", "active").Count(&n).Error
	return n, err
}

// CountCallsSince 自给定时间起的调用量.
func CountCallsSince(since time.Time) (int64, error) {
	var n int64
	err := DB.Model(&CallLog{}).Where("created_at >= ?", since).Count(&n).Error
	return n, err
}

// CountAllCalls 累计调用量.
func CountAllCalls() (int64, error) {
	var n int64
	err := DB.Model(&CallLog{}).Count(&n).Error
	return n, err
}

// ListCallLogs 按 Channel 与时间区间查询调用日志; channelID 必填, 区间可选, 时间口径 UTC.
func ListCallLogs(channelID uint, start, end *time.Time) ([]CallLog, int64, error) {
	q := DB.Model(&CallLog{}).Where("channel_id = ?", channelID)
	if start != nil {
		q = q.Where("created_at >= ?", *start)
	}
	if end != nil {
		q = q.Where("created_at < ?", end.AddDate(0, 0, 1))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var logs []CallLog
	if err := q.Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// BatchInsertCallLogs 单事务批量写入调用日志并回写涉及 Channel 的 last_used_at.
func BatchInsertCallLogs(logs []CallLog) error {
	if len(logs) == 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&logs).Error; err != nil {
			return err
		}
		last := make(map[uint]time.Time)
		for _, l := range logs {
			if l.ChannelID != nil && l.CreatedAt.After(last[*l.ChannelID]) {
				last[*l.ChannelID] = l.CreatedAt
			}
		}
		for id, at := range last {
			if err := tx.Model(&Channel{}).Where("id = ?", id).Update("last_used_at", at).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// IsUniqueViolation 判断是否唯一约束冲突 (source_url, name/slug, channel_id).
func IsUniqueViolation(err error) bool {
	return err != nil && errors.Is(err, gorm.ErrDuplicatedKey)
}
