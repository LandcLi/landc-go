package lock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/cache"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ============================================================
// RedisLock — 基于 Redis 的分布式锁
// 复用 frame/pkg/cache.GetRedis() 获取 Redis 客户端
// ============================================================

type RedisLock struct {
	client    redis.UniversalClient
	key       string
	value     string
	config    *LockConfig
	renewStop chan struct{}
	mu        sync.Mutex
	locked    bool
}

func NewRedisLock(name string, opts ...LockOption) *RedisLock {
	cfg := DefaultLockConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// 复用 frame/pkg/cache 获取 Redis 客户端
	redisClient := cache.GetRedis()
	if redisClient == nil {
		panic("workflow: Redis not initialized, call cache.InitGlobalCacheWithObject first")
	}

	return &RedisLock{
		client:    redisClient,
		key:       fmt.Sprintf("wf:lock:%s", name),
		value:     uuid.New().String(),
		config:    cfg,
		renewStop: make(chan struct{}),
	}
}

func (l *RedisLock) Lock(ctx context.Context) error {
	for i := 0; i < l.config.Retry; i++ {
		ok, err := l.tryLock(ctx)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(l.config.RetryWait):
		}
	}
	return fmt.Errorf("lock: failed to acquire lock after %d retries: %s", l.config.Retry, l.key)
}

func (l *RedisLock) TryLock(ctx context.Context) (bool, error) {
	return l.tryLock(ctx)
}

func (l *RedisLock) tryLock(ctx context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	ok, err := l.client.SetNX(ctx, l.key, l.value, l.config.TTL).Result()
	if err != nil {
		return false, err
	}
	if ok {
		l.locked = true
		if l.config.AutoRenew {
			l.startAutoRenew(ctx)
		}
	}
	return ok, nil
}

func (l *RedisLock) Unlock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.locked {
		return nil
	}
	defer func() { l.locked = false }()

	if l.config.AutoRenew {
		close(l.renewStop)
	}

	// Lua 脚本保证原子性
	script := `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
	`
	return l.client.Eval(ctx, script, []string{l.key}, l.value).Err()
}

func (l *RedisLock) startAutoRenew(ctx context.Context) {
	ticker := time.NewTicker(l.config.TTL / 3)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				l.mu.Lock()
				if !l.locked {
					l.mu.Unlock()
					return
				}
				script := `
				if redis.call("GET", KEYS[1]) == ARGV[1] then
					return redis.call("PEXPIRE", KEYS[1], ARGV[2])
				else
					return 0
				end
				`
				_ = l.client.Eval(ctx, script, []string{l.key}, l.value, l.config.TTL.Milliseconds()).Err()
				l.mu.Unlock()
			case <-l.renewStop:
				return
			}
		}
	}()
}
