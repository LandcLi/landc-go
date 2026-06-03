package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/config"
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
	globalCache  Cache
	globalRedis  *redis.Client
	cacheMu      sync.RWMutex
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

// InitGlobalCacheWithDefault 使用默认配置初始化
func InitGlobalCacheWithDefault() error {
	cfg := config.GetConfig()
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	return InitGlobalCacheWithConfig(&cfg.Redis)
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

// Close 关闭 Redis 连接
func Close() error {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if globalRedis == nil {
		return nil
	}

	err := globalRedis.Close()
	globalRedis = nil
	globalCache = nil
	return err
}

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
