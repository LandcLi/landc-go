package health

import (
	"context"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/cache"
)

type redisChecker struct{}

func (c *redisChecker) Name() string { return "redis" }

func (c *redisChecker) Check(ctx context.Context) error {
	redisClient := cache.GetRedis()
	if redisClient == nil {
		// Redis 未初始化（未配置或使用本地缓存），跳过检查
		return nil
	}

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return redisClient.Ping(pingCtx).Err()
}
