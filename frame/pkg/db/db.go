package db

import (
	"fmt"
	"sync"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/config"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	globalDB *gorm.DB
	dbMu     sync.RWMutex
)

// InitGlobalDBWithObject 使用已有的 GORM 实例初始化全局数据库
func InitGlobalDBWithObject(db *gorm.DB) {
	dbMu.Lock()
	defer dbMu.Unlock()
	globalDB = db
}

// InitGlobalDBWithConfig 使用配置初始化全局数据库
func InitGlobalDBWithConfig(cfg *config.DatabaseConfig) error {
	dbMu.Lock()
	defer dbMu.Unlock()

	if globalDB != nil {
		return nil
	}

	db, err := openDB(cfg)
	if err != nil {
		return err
	}

	globalDB = db
	return nil
}

// InitGlobalDBWithDefault 使用默认配置初始化全局数据库
func InitGlobalDBWithDefault() error {
	cfg := config.GetConfig()
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	return InitGlobalDBWithConfig(&cfg.Database)
}

// GetDB 获取全局数据库实例
func GetDB() *gorm.DB {
	dbMu.RLock()
	defer dbMu.RUnlock()
	return globalDB
}

// Close 关闭全局数据库连接
func Close() error {
	dbMu.Lock()
	defer dbMu.Unlock()

	if globalDB == nil {
		return nil
	}

	sqlDB, err := globalDB.DB()
	if err != nil {
		return err
	}
	globalDB = nil
	return sqlDB.Close()
}

// AutoMigrate 自动迁移数据表
func AutoMigrate(models ...interface{}) error {
	db := GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	return db.AutoMigrate(models...)
}

func openDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dialector, err := getDialector(cfg)
	if err != nil {
		return nil, err
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	}

	return db, nil
}

func getDialector(cfg *config.DatabaseConfig) (gorm.Dialector, error) {
	switch cfg.Driver {
	case "mysql":
		return getMySQLDialector(cfg)
	case "postgres", "postgresql":
		return getPostgresDialector(cfg)
	case "sqlite", "sqlite3":
		return getSQLiteDialector(cfg)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
}
