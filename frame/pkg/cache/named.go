package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/LandcLi/landc-go/frame/pkg/resource"
	toolscache "github.com/LandcLi/landc-go/tools/cache"
	"github.com/redis/go-redis/v9"
)

// namedCaches 保存命名缓存实例：name -> Cache。
var namedCaches sync.Map

// InitNamedCacheWithConfig 注册命名 Redis 缓存。
func InitNamedCacheWithConfig(name string, cfg *config.RedisConfig) error {
	if name == "" {
		return fmt.Errorf("cache: named cache name cannot be empty")
	}
	if cfg == nil {
		return fmt.Errorf("cache: named cache %q config cannot be nil", name)
	}

	cacheMu.Lock()
	defer cacheMu.Unlock()
	if _, ok := namedCaches.Load(name); ok {
		return nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("cache: init named cache %q: %w", name, err)
	}

	namedCaches.Store(name, &RedisCache{client: client})
	return nil
}

// InitNamedCacheWithLocal 注册命名本地内存缓存。
func InitNamedCacheWithLocal(name string, capacity int) {
	if name == "" {
		return
	}

	cacheMu.Lock()
	defer cacheMu.Unlock()
	if _, ok := namedCaches.Load(name); ok {
		return
	}

	gc := toolscache.NewGlobalCacheWithCapacity(capacity)
	gc.StartCleanup(10 * time.Minute)
	namedCaches.Store(name, &LocalCache{cache: gc})
}

// HasNamedCache 报告命名缓存是否已注册。
func HasNamedCache(name string) bool {
	_, ok := namedCaches.Load(name)
	return ok
}

// GetNamedCache 返回命名缓存；未注册返回 nil。
func GetNamedCache(name string) Cache {
	v, ok := namedCaches.Load(name)
	if !ok {
		return nil
	}
	return v.(Cache)
}

// GetCacheFrom 从 ctx 解析命名缓存。
// 作用域指定了 Cache 且命名缓存存在时返回命名缓存；否则回退全局缓存。
// 作用域指定的命名缓存未注册时返回 nil（注册入口已校验，运行时 nil 属边界情况）。
func GetCacheFrom(ctx context.Context) Cache {
	if s, ok := resource.FromContext(ctx); ok && s.Cache != "" {
		if c := GetNamedCache(s.Cache); c != nil {
			return c
		}
		return nil
	}
	return GetCache()
}
