// Package store 是 RandImg 的唯一数据入口: 四实体表, schema 迁移与查询函数集.
// 模块定义见 .trellis/spec/arch/store.md.
package store

import (
	"fmt"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库连接并迁移 schema.
func InitDB(dbPath string) error {
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Warn),
		TranslateError: true,
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	if err := AutoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("[store] database initialized:", dbPath)
	return nil
}

// AutoMigrate 建表迁移. api_keys 等旧表不迁移, 部署侧可删库文件干净重建.
func AutoMigrate() error {
	return DB.AutoMigrate(
		&Category{},
		&Image{},
		&Channel{},
		&CallLog{},
		&Setting{},
	)
}
