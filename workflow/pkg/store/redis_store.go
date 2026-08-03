package store

import (
	"context"
	"fmt"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/cache"
)

// ============================================================
// RedisStore — 基于 frame/pkg/cache 的辅助存储
// 复用 frame/pkg/cache.Cache 接口（支持 Redis / 本地 LRU 自动切换）
// ============================================================

type RedisStore struct {
	cacheClient cache.Cache
	prefix      string
}

func NewRedisStore(prefix string) *RedisStore {
	if prefix == "" {
		prefix = "wf"
	}
	cc := cache.GetCache()
	if cc == nil {
		panic("workflow: cache not initialized, call cache.InitGlobalCacheWithDefault first")
	}
	return &RedisStore{cacheClient: cc, prefix: prefix}
}

func (s *RedisStore) key(parts ...string) string {
	result := s.prefix
	for _, p := range parts {
		result += ":" + p
	}
	return result
}

// ==================== 执行缓存 ====================

func (s *RedisStore) CacheExecution(ctx context.Context, execID string, data interface{}, ttl time.Duration) error {
	return s.cacheClient.SetObject(ctx, s.key("exec", execID), data, ttl)
}

func (s *RedisStore) GetCachedExecution(ctx context.Context, execID string, dest interface{}) error {
	return s.cacheClient.GetObject(ctx, s.key("exec", execID), dest)
}

// ==================== 幂等性 ====================

func (s *RedisStore) IsAttemptProcessed(ctx context.Context, attemptID string) (bool, error) {
	return s.cacheClient.Exists(ctx, s.key("attempt", attemptID))
}

func (s *RedisStore) MarkAttemptProcessed(ctx context.Context, attemptID string, ttl time.Duration) error {
	return s.cacheClient.Set(ctx, s.key("attempt", attemptID), "1", ttl)
}

// ==================== Worker 心跳 ====================

func (s *RedisStore) WorkerHeartbeat(ctx context.Context, workerID string, ttl time.Duration) error {
	return s.cacheClient.Set(ctx, s.key("worker", workerID), time.Now().Unix(), ttl)
}

func (s *RedisStore) IsWorkerAlive(ctx context.Context, workerID string) (bool, error) {
	return s.cacheClient.Exists(ctx, s.key("worker", workerID))
}

// ==================== 分布式计数器 ====================

func (s *RedisStore) IncrCounter(ctx context.Context, name string) (int64, error) {
	// frame/pkg/cache 不直接提供 Incr, 需要降级到本地处理
	// 实际使用时可从 cache.GetRedis() 获取原生 Redis 客户端
	return 0, fmt.Errorf("incr not supported via Cache interface, use raw Redis")
}
