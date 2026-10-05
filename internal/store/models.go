package store

import "time"

// Category 分类表.
type Category struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string `gorm:"type:varchar(50);not null;uniqueIndex" json:"name"`
	Slug        string `gorm:"type:varchar(50);not null;uniqueIndex" json:"slug"`
	Description string `gorm:"type:text" json:"description"`
}

func (Category) TableName() string { return "categories" }

// Image 图片记录; Width/Height/Format 为空表示待元数据补全.
type Image struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	SourceURL  string    `gorm:"type:text;not null;uniqueIndex" json:"source_url"`
	Width      *int      `gorm:"type:integer" json:"width"`
	Height     *int      `gorm:"type:integer" json:"height"`
	Format     string    `gorm:"type:varchar(10)" json:"format"`
	Source     string    `gorm:"type:varchar(255)" json:"source"`
	Status     string    `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	FetchFails uint      `gorm:"not null;default:0" json:"fetch_fails"` // 元数据补全连续失败次数; >=3 不再重扫
	CategoryID uint      `gorm:"not null;index" json:"category_id"`
	Category   *Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Image) TableName() string { return "images" }

// Channel 公开接入标识: 流量归属, 配额与熔断的载体, 不承担保密职责.
// AllowedOrigins 为 JSON 字符串数组; 空串表示不校验来源.
type Channel struct {
	ID             uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	ChannelID      string     `gorm:"type:varchar(64);not null;uniqueIndex" json:"channel_id"`
	RateLimit      int        `gorm:"not null;default:60" json:"rate_limit"`
	Status         string     `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	AllowedOrigins string     `gorm:"type:text" json:"allowed_origins"`
	CreatedAt      time.Time  `json:"created_at"`
	LastUsedAt     *time.Time `json:"last_used_at"`
}

func (Channel) TableName() string { return "channels" }

// CallLog 公开面调用的异步计量记录; ChannelID 为空表示匿名调用.
type CallLog struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ChannelID *uint     `gorm:"index" json:"channel_id"`
	Path      string    `gorm:"type:varchar(255);not null" json:"path"`
	CreatedAt time.Time `gorm:"not null;index" json:"created_at"`
}

func (CallLog) TableName() string { return "call_logs" }
