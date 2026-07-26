package lock

import "github.com/LandcLi/landc-go/frame/pkg/cache"

// RedisLockFactory — 复用 frame/pkg/cache 获取 Redis 客户端
type RedisLockFactory struct{}

func NewRedisLockFactory() *RedisLockFactory {
	// 确保 Redis 已初始化
	if cache.GetRedis() == nil {
		panic("workflow: Redis not initialized, call cache.InitGlobalCacheWithObject first")
	}
	return &RedisLockFactory{}
}

func (f *RedisLockFactory) NewLock(name string, opts ...LockOption) DistributedLock {
	return NewRedisLock(name, opts...)
}
