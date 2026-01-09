package database

import (
	"fmt"
	"log"
	"randimg/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB(dbPath string) error {
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	// 自动迁移
	if err := AutoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate() error {
	return DB.AutoMigrate(
		&model.Category{},
		&model.Image{},
		&model.APIKey{},
		&model.APIUsageLog{},
	)
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}
