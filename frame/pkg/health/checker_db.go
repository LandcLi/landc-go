package health

import (
	"context"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/db"
)

type dbChecker struct {
	onlyIfConfigured bool
}

func (c *dbChecker) Name() string { return "database" }

func (c *dbChecker) Check(ctx context.Context) error {
	gormDB := db.GetDB()
	if gormDB == nil {
		// 数据库未初始化（未配置），跳过检查
		return nil
	}

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(pingCtx)
}
