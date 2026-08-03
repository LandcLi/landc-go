package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/redis/go-redis/v9"
)

func setupTestRedis(t *testing.T) *RedisCache {
	t.Helper()

	cfg := &config.RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       9, // 使用测试专用 DB
		PoolSize: 10,
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// 添加超时，避免 Redis 不可用时测试卡住
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping test")
	}

	InitGlobalCacheWithObject(client)

	return GetCache().(*RedisCache)
}

func cleanupRedis(t *testing.T, keys ...string) {
	t.Helper()
	cache := GetCache()
	if cache == nil {
		return
	}
	ctx := context.Background()
	for _, key := range keys {
		cache.Delete(ctx, key)
	}
}

func TestInitGlobalCacheWithObject(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   9,
	})

	InitGlobalCacheWithObject(client)

	cache := GetCache()
	if cache == nil {
		t.Error("GetCache should not return nil")
	}

	redisClient := GetRedis()
	if redisClient == nil {
		t.Error("GetRedis should not return nil")
	}

	Close()
}

func TestInitGlobalCacheWithConfig(t *testing.T) {
	cfg := &config.RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       9,
		PoolSize: 10,
	}

	err := InitGlobalCacheWithConfig(cfg)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}

	cache := GetCache()
	if cache == nil {
		t.Error("GetCache should not return nil")
	}

	Close()
}

func TestInitGlobalCacheWithDefault(t *testing.T) {
	err := InitGlobalCacheWithDefault()
	if err != nil {
		t.Skip("Config not initialized, skipping test")
	}

	cache := GetCache()
	if cache == nil {
		t.Error("GetCache should not return nil")
	}

	Close()
}

func TestRedisCache_GetSet(t *testing.T) {
	cache := setupTestRedis(t)
	defer Close()
	defer cleanupRedis(t, "test_key")

	ctx := context.Background()

	err := cache.Set(ctx, "test_key", "test_value", 10*time.Second)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := cache.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if val != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", val)
	}
}

func TestRedisCache_Delete(t *testing.T) {
	cache := setupTestRedis(t)
	defer Close()
	defer cleanupRedis(t, "test_key")

	ctx := context.Background()

	cache.Set(ctx, "test_key", "test_value", 10*time.Second)

	err := cache.Delete(ctx, "test_key")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = cache.Get(ctx, "test_key")
	if err == nil {
		t.Error("Get should fail after delete")
	}
}

func TestRedisCache_Exists(t *testing.T) {
	cache := setupTestRedis(t)
	defer Close()
	defer cleanupRedis(t, "test_key")

	ctx := context.Background()

	exists, err := cache.Exists(ctx, "test_key")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("Key should not exist")
	}

	cache.Set(ctx, "test_key", "test_value", 10*time.Second)

	exists, err = cache.Exists(ctx, "test_key")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Key should exist")
	}
}

func TestRedisCache_Expire(t *testing.T) {
	cache := setupTestRedis(t)
	defer Close()
	defer cleanupRedis(t, "test_key")

	ctx := context.Background()

	cache.Set(ctx, "test_key", "test_value", 10*time.Second)

	err := cache.Expire(ctx, "test_key", 1*time.Second)
	if err != nil {
		t.Fatalf("Expire failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	_, err = cache.Get(ctx, "test_key")
	if err == nil {
		t.Error("Key should be expired")
	}
}

func TestRedisCache_GetObject(t *testing.T) {
	cache := setupTestRedis(t)
	defer Close()
	defer cleanupRedis(t, "test_obj")

	ctx := context.Background()

	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	obj := &TestStruct{Name: "Alice", Age: 30}

	err := cache.SetObject(ctx, "test_obj", obj, 10*time.Second)
	if err != nil {
		t.Fatalf("SetObject failed: %v", err)
	}

	var retrieved TestStruct
	err = cache.GetObject(ctx, "test_obj", &retrieved)
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	}

	if retrieved.Name != "Alice" || retrieved.Age != 30 {
		t.Errorf("Expected {Alice 30}, got {%s %d}", retrieved.Name, retrieved.Age)
	}
}

func TestRedisCache_SetObject(t *testing.T) {
	cache := setupTestRedis(t)
	defer Close()
	defer cleanupRedis(t, "test_obj")

	ctx := context.Background()

	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	obj := &TestStruct{Name: "Bob", Age: 25}

	err := cache.SetObject(ctx, "test_obj", obj, 10*time.Second)
	if err != nil {
		t.Fatalf("SetObject failed: %v", err)
	}

	var retrieved TestStruct
	err = cache.GetObject(ctx, "test_obj", &retrieved)
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	}

	if retrieved.Name != "Bob" || retrieved.Age != 25 {
		t.Errorf("Expected {Bob 25}, got {%s %d}", retrieved.Name, retrieved.Age)
	}
}

func TestGetCacheThreadSafety(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   9,
	})

	InitGlobalCacheWithObject(client)
	defer Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache := GetCache()
			if cache == nil {
				t.Error("GetCache should not return nil in concurrent access")
			}
		}()
	}
	wg.Wait()
}

func TestClose(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   9,
	})

	InitGlobalCacheWithObject(client)

	err := Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	cache := GetCache()
	if cache != nil {
		t.Error("GetCache should return nil after Close")
	}
}

func TestRedisCache_Integration(t *testing.T) {
	cache := setupTestRedis(t)
	defer Close()
	defer cleanupRedis(t, "key1", "key2", "key3", "obj1")

	ctx := context.Background()

	cache.Set(ctx, "key1", "value1", 10*time.Second)
	cache.Set(ctx, "key2", "value2", 10*time.Second)

	val1, _ := cache.Get(ctx, "key1")
	val2, _ := cache.Get(ctx, "key2")

	if val1 != "value1" || val2 != "value2" {
		t.Error("Set/Get values mismatch")
	}

	type Data struct {
		ID   int    `json:"id"`
		Info string `json:"info"`
	}

	cache.SetObject(ctx, "obj1", &Data{ID: 1, Info: "test"}, 10*time.Second)

	var data Data
	cache.GetObject(ctx, "obj1", &data)

	if data.ID != 1 || data.Info != "test" {
		t.Error("SetObject/GetObject values mismatch")
	}

	cache.Delete(ctx, "key1")
	_, err := cache.Get(ctx, "key1")
	if err == nil {
		t.Error("key1 should be deleted")
	}
}

func TestInitGlobalCacheWithConfig_DuplicateInit(t *testing.T) {
	cfg := &config.RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       9,
	}

	err := InitGlobalCacheWithConfig(cfg)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}

	err = InitGlobalCacheWithConfig(cfg)
	if err != nil {
		t.Fatalf("Second init should not fail (should skip): %v", err)
	}

	Close()
}
