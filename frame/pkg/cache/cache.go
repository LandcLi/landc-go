package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/config"
	toolscache "github.com/LandcLi/landc-go/tools/cache"
	"github.com/redis/go-redis/v9"
)

// Cache 缓存接口
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	GetObject(ctx context.Context, key string, dest interface{}) error
	SetObject(ctx context.Context, key string, value interface{}, expiration time.Duration) error
}

var (
	globalCache Cache
	globalRedis *redis.Client
	cacheMu     sync.RWMutex
)

// InitGlobalCacheWithObject 使用已有的 Redis 客户端初始化
func InitGlobalCacheWithObject(client *redis.Client) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	globalRedis = client
	globalCache = &RedisCache{client: client}
}

// InitGlobalCacheWithConfig 使用配置初始化
func InitGlobalCacheWithConfig(cfg *config.RedisConfig) error {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if globalCache != nil {
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
		return fmt.Errorf("failed to connect redis: %w", err)
	}

	globalRedis = client
	globalCache = &RedisCache{client: client}
	return nil
}

// InitGlobalCacheWithLocal 使用本地内存缓存初始化（基于 tools/cache LRU 实现）
func InitGlobalCacheWithLocal(capacity int) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if globalCache != nil {
		return
	}

	gc := toolscache.NewGlobalCacheWithCapacity(capacity)
	gc.StartCleanup(10 * time.Minute)
	globalCache = &LocalCache{cache: gc}
}

// InitGlobalCacheWithDefault 使用默认配置初始化：
// 如果有 Redis 配置且可连接，使用 Redis；否则使用本地内存缓存（弱依赖模式）
func InitGlobalCacheWithDefault() error {
	cfg := config.GetConfig()
	if cfg == nil {
		InitGlobalCacheWithLocal(10000)
		return nil
	}

	// 尝试 Redis
	if cfg.Redis.Addr != "" && cfg.Redis.Addr != "localhost:6379" {
		err := InitGlobalCacheWithConfig(&cfg.Redis)
		if err == nil {
			return nil
		}
	}

	// Redis 不可用或未配置，使用本地缓存
	InitGlobalCacheWithLocal(10000)
	return nil
}

// GetCache 获取缓存实例
func GetCache() Cache {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return globalCache
}

// GetRedis 获取 Redis 客户端
func GetRedis() *redis.Client {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return globalRedis
}

// Close 关闭缓存连接
func Close() error {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if globalRedis != nil {
		err := globalRedis.Close()
		globalRedis = nil
		globalCache = nil
		return err
	}

	// 本地缓存关闭（清理 goroutine）
	if lc, ok := globalCache.(*LocalCache); ok {
		lc.cache.StopCleanup()
	}
	globalCache = nil
	return nil
}

// ==================== RedisCache ====================

// RedisCache Redis 缓存实现
type RedisCache struct {
	client *redis.Client
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

func (c *RedisCache) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (c *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

func (c *RedisCache) GetObject(ctx context.Context, key string, dest interface{}) error {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

func (c *RedisCache) SetObject(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, expiration).Err()
}

// ==================== LocalCache ====================

// LocalCache 基于 tools/cache LRU 实现的本地缓存（与 Cache 接口兼容）
type LocalCache struct {
	cache *toolscache.GlobalCache
}

func (c *LocalCache) Get(ctx context.Context, key string) (string, error) {
	val, found := c.cache.Get(key)
	if !found {
		return "", errors.New("key not found")
	}
	switch v := val.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		data, err := json.Marshal(val)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

func (c *LocalCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	c.cache.SetWithExpiration(key, value, expiration)
	return nil
}

func (c *LocalCache) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		c.cache.Delete(key)
	}
	return nil
}

func (c *LocalCache) Exists(ctx context.Context, key string) (bool, error) {
	_, found := c.cache.Get(key)
	return found, nil
}

func (c *LocalCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	val, found := c.cache.Get(key)
	if !found {
		return errors.New("key not found")
	}
	c.cache.SetWithExpiration(key, val, expiration)
	return nil
}

func (c *LocalCache) GetObject(ctx context.Context, key string, dest interface{}) error {
	val, found := c.cache.Get(key)
	if !found {
		return errors.New("key not found")
	}
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (c *LocalCache) SetObject(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	c.cache.SetWithExpiration(key, value, expiration)
	return nil
}
