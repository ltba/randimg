package store

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// CountCallsSince 自给定时间起的 Channel 调用量; 口径: 按日聚合表, 匿名不计入.
func CountCallsSince(since time.Time) (int64, error) {
	var n int64
	err := DB.Model(&DailyCall{}).
		Where("channel_id > 0 AND date >= ?", since.UTC().Format("2006-01-02")).
		Select("COALESCE(SUM(count), 0)").Scan(&n).Error
	return n, err
}

// CountAllCalls 累计 Channel 调用量; 口径: 按日聚合表, 匿名不计入.
func CountAllCalls() (int64, error) {
	var n int64
	err := DB.Model(&DailyCall{}).
		Where("channel_id > 0").
		Select("COALESCE(SUM(count), 0)").Scan(&n).Error
	return n, err
}

// CountAnonCallsSince 自给定时间起的匿名调用量; 与总计数口径分离, 单独计量.
func CountAnonCallsSince(since time.Time) (int64, error) {
	var n int64
	err := DB.Model(&DailyCall{}).
		Where("channel_id = 0 AND date >= ?", since.UTC().Format("2006-01-02")).
		Select("COALESCE(SUM(count), 0)").Scan(&n).Error
	return n, err
}

// CountAnonAllCalls 累计匿名调用量; 与总计数口径分离, 单独计量.
func CountAnonAllCalls() (int64, error) {
	var n int64
	err := DB.Model(&DailyCall{}).
		Where("channel_id = 0").
		Select("COALESCE(SUM(count), 0)").Scan(&n).Error
	return n, err
}

// DailyCallDelta 单次批量落库的按日聚合增量.
type DailyCallDelta struct {
	Date      string
	ChannelID uint
	N         int64
}

// FlushCallLogs 单事务写入调用明细, 累加按日聚合, 并回写涉及 Channel 的 last_used_at; 计量落库唯一入口.
func FlushCallLogs(logs []CallLog, deltas []DailyCallDelta) error {
	if len(logs) == 0 && len(deltas) == 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if len(logs) > 0 {
			if err := tx.Create(&logs).Error; err != nil {
				return err
			}
		}
		for _, d := range deltas {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "date"}, {Name: "channel_id"}},
				DoUpdates: clause.Assignments(map[string]interface{}{"count": gorm.Expr("count + ?", d.N)}),
			}).Create(&DailyCall{Date: d.Date, ChannelID: d.ChannelID, Count: d.N}).Error; err != nil {
				return err
			}
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

// BackfillDailyCalls 首次迁移: call_logs 全量聚合回填 daily_calls (仅当聚合表为空).
func BackfillDailyCalls() error {
	var n int64
	if err := DB.Model(&DailyCall{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	type aggRow struct {
		Date      string
		ChannelID uint
		Count     int64
	}
	var rows []aggRow
	if err := DB.Model(&CallLog{}).
		Select("strftime('%Y-%m-%d', created_at) AS date, COALESCE(channel_id, 0) AS channel_id, COUNT(*) AS count").
		Group("date, channel_id").Scan(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	batch := make([]DailyCall, len(rows))
	for i, r := range rows {
		batch[i] = DailyCall{Date: r.Date, ChannelID: r.ChannelID, Count: r.Count}
	}
	return DB.CreateInBatches(&batch, 100).Error
}

// DeleteCallLogsBefore 删除指定时间前的调用明细, 返回删除行数.
func DeleteCallLogsBefore(t time.Time) (int64, error) {
	res := DB.Where("created_at < ?", t).Delete(&CallLog{})
	return res.RowsAffected, res.Error
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

// IsUniqueViolation 判断是否唯一约束冲突 (source_url, name/slug, channel_id).
func IsUniqueViolation(err error) bool {
	return err != nil && errors.Is(err, gorm.ErrDuplicatedKey)
}

// GetSettingValue 读配置值; 无记录返回空串.
func GetSettingValue(key string) (string, error) {
	var s Setting
	err := DB.Where("key = ?", key).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return s.Value, nil
}

// SetSettingValue 写配置值 (upsert).
func SetSettingValue(key, value string) error {
	return DB.Save(&Setting{Key: key, Value: value}).Error
}
