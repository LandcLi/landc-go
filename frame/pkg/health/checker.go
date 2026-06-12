// Package health 提供框架内置的 DB/Redis 健康检查器。
//
// 本包是 api/health 框架特有实现的补充，依赖 frame/pkg/db 和 frame/pkg/cache。
// 通用的 Checker 接口、类型、注册表在 api/health 包中。
package health

import (
	"context"
	"time"

	apihealth "github.com/LandcLi/landc-go/api/health"
	"github.com/LandcLi/landc-go/frame/pkg/cache"
	"github.com/LandcLi/landc-go/frame/pkg/db"
)

// RegisterDefaultCheckers 注册框架内置的 DB 和 Redis 健康检查器。
func RegisterDefaultCheckers(dbEnabled, redisEnabled bool) {
	if dbEnabled {
		apihealth.Register(&dbChecker{})
	}
	if redisEnabled {
		apihealth.Register(&redisChecker{})
	}
}

// dbChecker 数据库健康检查器。
type dbChecker struct{}

func (c *dbChecker) Name() string { return "database" }

func (c *dbChecker) Check(ctx context.Context) error {
	gormDB := db.GetDB()
	if gormDB == nil {
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

// redisChecker Redis 健康检查器。
type redisChecker struct{}

func (c *redisChecker) Name() string { return "redis" }

func (c *redisChecker) Check(ctx context.Context) error {
	redisClient := cache.GetRedis()
	if redisClient == nil {
		return nil
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return redisClient.Ping(pingCtx).Err()
}
