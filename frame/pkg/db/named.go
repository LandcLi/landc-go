package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/LandcLi/landc-go/frame/pkg/resource"
	"gorm.io/gorm"
)

// namedDBs 保存命名数据库连接：name -> *gorm.DB。
var namedDBs sync.Map

// InitNamedDB 注册命名数据库连接。
// 供库模式嵌入场景使用：宿主可为嵌入服务准备独立的数据库连接。
func InitNamedDB(name string, cfg *config.DatabaseConfig) error {
	if name == "" {
		return fmt.Errorf("db: named db name cannot be empty")
	}
	if cfg == nil {
		return fmt.Errorf("db: named db %q config cannot be nil", name)
	}

	dbMu.Lock()
	defer dbMu.Unlock()
	if _, ok := namedDBs.Load(name); ok {
		return nil
	}

	gdb, err := openDB(cfg)
	if err != nil {
		return fmt.Errorf("db: init named db %q: %w", name, err)
	}
	namedDBs.Store(name, gdb)
	return nil
}

// HasNamedDB 报告命名连接是否已注册。
func HasNamedDB(name string) bool {
	_, ok := namedDBs.Load(name)
	return ok
}

// GetNamedDB 返回命名连接；未注册返回 nil。
func GetNamedDB(name string) *gorm.DB {
	v, ok := namedDBs.Load(name)
	if !ok {
		return nil
	}
	return v.(*gorm.DB)
}

// GetDBFrom 从 ctx 解析命名连接。
// 作用域指定了 DB 且命名连接存在时返回命名连接；否则回退全局连接。
// 作用域指定的命名连接未注册时返回 nil（注册入口已校验，运行时 nil 属资源被关闭等边界情况）。
func GetDBFrom(ctx context.Context) *gorm.DB {
	if s, ok := resource.FromContext(ctx); ok && s.DB != "" {
		if d := GetNamedDB(s.DB); d != nil {
			return d
		}
		return nil
	}
	return globalDB
}
