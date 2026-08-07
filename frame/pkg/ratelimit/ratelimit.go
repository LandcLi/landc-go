// Package ratelimit 提供基于缓存的限流（间隔限流 / 计数限流）。
//
// 与 tools/ratelimit 的关系：本包是 frame 层包装——缓存从请求 ctx 解析
// （framecache.GetCacheFrom），因此：
//   - 独立部署使用全局缓存
//   - 库模式嵌入（web.RegisterLibrary + WithScope）自动使用命名缓存
//
// 缓存不可用时放行（fail-open），避免缓存故障误伤正常业务。
package ratelimit

import (
	"context"
	"time"

	framecache "github.com/LandcLi/landc-go/frame/pkg/cache"
	toolsrl "github.com/LandcLi/landc-go/tools/ratelimit"
)

// AllowInterval 间隔限流：同一 key 在 interval 内只放行一次。
// 典型场景：验证码发送间隔 60s。
func AllowInterval(ctx context.Context, key string, interval time.Duration) bool {
	c := framecache.GetCacheFrom(ctx)
	if c == nil {
		return true
	}
	return toolsrl.NewIntervalLimiter(framecache.AsToolsCache(c)).Allow(key, interval)
}

// AllowCount 计数限流：同一 key 在 window 内最多放行 limit 次。
// 计数经 Cache.Incr 原子自增，并发下不丢计数。
// 典型场景：每日发送上限。
func AllowCount(ctx context.Context, key string, limit int64, window time.Duration) bool {
	c := framecache.GetCacheFrom(ctx)
	if c == nil {
		return true
	}
	return toolsrl.NewCountLimiter(framecache.AsToolsCache(c)).Allow(key, limit, window)
}
