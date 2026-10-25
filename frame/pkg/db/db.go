package db

import (
	"errors"
	"fmt"
	"os"
	"strings"
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

// ErrNoDatabaseConfig 表示配置中未指定数据库驱动，属于「不需要数据库」的正常场景，
// 与「配置了数据库但连接失败」必须区分开：前者可跳过，后者应快速失败。
var ErrNoDatabaseConfig = errors.New("database driver not configured")

// InitGlobalDBWithDefault 使用默认配置初始化全局数据库
func InitGlobalDBWithDefault() error {
	cfg := config.GetConfig()
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	if strings.TrimSpace(cfg.Database.Driver) == "" {
		return ErrNoDatabaseConfig
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

// AutoMigrateIfEnabled 在配置开启 database.auto_migrate 时执行 AutoMigrate，
// 否则直接返回 nil（不视为错误）。
//
// 用途：让业务系统用一行代码完成「开发/测试环境自动建表」，生产环境仍由
// sqls/*.sql 或版本化迁移掌控，两种模式共存且无需改代码：
//
//	if err := db.AutoMigrateIfEnabled(&model.User{}, &model.LoginHistory{}); err != nil {
//	    return err
//	}
func AutoMigrateIfEnabled(models ...interface{}) error {
	cfg := config.GetConfig()
	if cfg == nil || !cfg.Database.AutoMigrate {
		return nil
	}
	if len(models) == 0 {
		return nil
	}
	return AutoMigrate(models...)
}

func openDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dialector, err := getDialector(cfg)
	if err != nil {
		return nil, err
	}

	// 生产环境默认 Warn 级别，避免输出全部 SQL 造成数据泄露与海量日志。
	// 可通过环境变量 LANDC_DB_LOG_MODE 覆盖（可选值：silent|error|warn|info）
	logMode := logger.Warn
	switch strings.ToLower(os.Getenv("LANDC_DB_LOG_MODE")) {
	case "silent":
		logMode = logger.Silent
	case "error":
		logMode = logger.Error
	case "warn":
		logMode = logger.Warn
	case "info":
		logMode = logger.Info
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logMode),
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
