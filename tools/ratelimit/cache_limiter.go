package ratelimit

import "time"

// Cache 是缓存限流依赖的存储接口（由调用方注入，本包不依赖任何框架）。
//
// Incr 必须为原子自增（并发下保证计数不丢），Redis 可用 INCR+EXPIRE，
// 内存实现需加锁。Get/Set/Exists 用于间隔限流的标记读写。
type Cache interface {
	Get(key string) (string, error)
	Set(key, value string, ttl time.Duration) error
	Delete(key string) error
	Exists(key string) (bool, error)
	// Incr 原子自增 key 并设置/刷新 TTL，返回自增后的值。
	Incr(key string, ttl time.Duration) (int64, error)
}

// IntervalLimiter 间隔限流：同一 key 在 interval 内只放行一次。
// 典型场景：验证码发送间隔 60s。
type IntervalLimiter struct {
	cache Cache
}

// NewIntervalLimiter 创建间隔限流器。
func NewIntervalLimiter(c Cache) *IntervalLimiter {
	return &IntervalLimiter{cache: c}
}

// Allow 返回 true 表示放行（窗口内首次）；窗口内的重复 key 被拒绝。
// 缓存异常时放行（fail-open），避免缓存故障误伤正常业务。
func (l *IntervalLimiter) Allow(key string, interval time.Duration) bool {
	exists, err := l.cache.Exists(key)
	if err != nil {
		return true
	}
	if exists {
		return false
	}
	if err := l.cache.Set(key, "1", interval); err != nil {
		return true
	}
	return true
}

// CountLimiter 计数限流：同一 key 在 window 内最多放行 limit 次。
// 典型场景：每日发送上限。
type CountLimiter struct {
	cache Cache
}

// NewCountLimiter 创建计数限流器。
func NewCountLimiter(c Cache) *CountLimiter {
	return &CountLimiter{cache: c}
}

// Allow 返回 true 表示放行（当前计数未超 limit）。
// 计数通过 Cache.Incr 原子自增，并发下不丢计数。
func (l *CountLimiter) Allow(key string, limit int64, window time.Duration) bool {
	n, err := l.cache.Incr(key, window)
	if err != nil {
		return true
	}
	return n <= limit
}
