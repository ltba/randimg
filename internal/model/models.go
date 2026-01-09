package model

import (
	"time"
)

// Category 分类表
type Category struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string `gorm:"type:varchar(50);not null;uniqueIndex" json:"name"`
	Slug        string `gorm:"type:varchar(50);not null;uniqueIndex" json:"slug"`
	Description string `gorm:"type:text" json:"description"`
}

// Image 图片信息表
type Image struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	SourceURL  string    `gorm:"type:text;not null;uniqueIndex" json:"source_url"`
	Width      *int      `gorm:"type:integer" json:"width"`
	Height     *int      `gorm:"type:integer" json:"height"`
	Format     string    `gorm:"type:varchar(10)" json:"format"`
	Source     string    `gorm:"type:varchar(255)" json:"source"`
	Status     string    `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	CategoryID uint      `gorm:"not null;index" json:"category_id"`
	Category   *Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// APIKey API密钥表
type APIKey struct {
	ID         uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Key        string     `gorm:"type:varchar(255);not null;uniqueIndex" json:"key"`
	UserID     *uint      `gorm:"type:integer" json:"user_id"`
	RateLimit  int        `gorm:"type:integer;not null;default:60" json:"rate_limit"`
	Status     string     `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

// APIUsageLog API调用日志表（简化版）
type APIUsageLog struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	APIKeyID    uint      `gorm:"not null;index" json:"api_key_id"`
	RequestedAt time.Time `gorm:"not null;index" json:"requested_at"`
}

// TableName 指定表名
func (Category) TableName() string {
	return "categories"
}

func (Image) TableName() string {
	return "images"
}

func (APIKey) TableName() string {
	return "api_keys"
}

func (APIUsageLog) TableName() string {
	return "api_usage_logs"
}
