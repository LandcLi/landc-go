package cache

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestGlobalCache_SetAndGet(t *testing.T) {
	cache := NewGlobalCache()

	cache.Set("key1", "value1")
	value, found := cache.Get("key1")

	if !found {
		t.Error("Expected to find key1")
	}

	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}
}

func TestGlobalCache_SetWithExpiration(t *testing.T) {
	cache := NewGlobalCache()

	cache.SetWithExpiration("key1", "value1", 100*time.Millisecond)
	value, found := cache.Get("key1")

	if !found {
		t.Error("Expected to find key1 immediately")
	}

	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	time.Sleep(150 * time.Millisecond)

	_, found = cache.Get("key1")
	if found {
		t.Error("Expected key1 to be expired")
	}
}

func TestGlobalCache_Delete(t *testing.T) {
	cache := NewGlobalCache()

	cache.Set("key1", "value1")
	cache.Delete("key1")

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be deleted")
	}
}

func TestGlobalCache_Clear(t *testing.T) {
	cache := NewGlobalCache()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0, got %d", cache.Size())
	}
}

func TestGlobalCache_Size(t *testing.T) {
	cache := NewGlobalCache()

	if cache.Size() != 0 {
		t.Errorf("Expected initial size 0, got %d", cache.Size())
	}

	cache.Set("key1", "value1")
	if cache.Size() != 1 {
		t.Errorf("Expected size 1, got %d", cache.Size())
	}

	cache.Set("key2", "value2")
	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}
}

func TestGlobalCache_Keys(t *testing.T) {
	cache := NewGlobalCache()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	keys := cache.Keys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	if !keyMap["key1"] || !keyMap["key2"] || !keyMap["key3"] {
		t.Error("Expected keys key1, key2, key3")
	}
}

func TestGlobalCache_ConcurrentAccess(t *testing.T) {
	cache := NewGlobalCache()
	var wg sync.WaitGroup

	numGoroutines := 100
	numOperations := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := string(rune('a' + (j % 26)))
				cache.Set(key, id)
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()

	expectedSize := 26
	if cache.Size() != expectedSize {
		t.Errorf("Expected cache size %d, got %d", expectedSize, cache.Size())
	}
}

func TestGlobalCache_CleanupExpired(t *testing.T) {
	cache := NewGlobalCache()

	cache.Set("permanent", "value")
	cache.SetWithExpiration("expired1", "value1", 50*time.Millisecond)
	cache.SetWithExpiration("expired2", "value2", 50*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	cache.CleanupExpired()

	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1 after cleanup, got %d", cache.Size())
	}

	_, found := cache.Get("permanent")
	if !found {
		t.Error("Expected permanent key to still exist")
	}

	_, found = cache.Get("expired1")
	if found {
		t.Error("Expected expired1 to be cleaned up")
	}

	_, found = cache.Get("expired2")
	if found {
		t.Error("Expected expired2 to be cleaned up")
	}
}

func TestGlobalCache_StartCleanup(t *testing.T) {
	cache := NewGlobalCache()

	cache.Set("permanent", "value")
	cache.SetWithExpiration("expired", "value", 50*time.Millisecond)

	cache.StartCleanup(50 * time.Millisecond)

	time.Sleep(150 * time.Millisecond)

	_, found := cache.Get("permanent")
	if !found {
		t.Error("Expected permanent key to still exist")
	}

	_, found = cache.Get("expired")
	if found {
		t.Error("Expected expired key to be cleaned up automatically")
	}
}

func TestGlobalCache_Capacity(t *testing.T) {
	cache := NewGlobalCacheWithCapacity(3)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	if cache.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cache.Size())
	}

	cache.Set("key4", "value4")

	if cache.Size() != 3 {
		t.Errorf("Expected size 3 after adding 4th item, got %d", cache.Size())
	}

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be evicted (LRU)")
	}

	_, found = cache.Get("key2")
	if !found {
		t.Error("Expected key2 to exist")
	}

	_, found = cache.Get("key3")
	if !found {
		t.Error("Expected key3 to exist")
	}

	_, found = cache.Get("key4")
	if !found {
		t.Error("Expected key4 to exist")
	}
}

func TestGlobalCache_LRU(t *testing.T) {
	cache := NewGlobalCacheWithCapacity(3)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	cache.Get("key1")

	cache.Set("key4", "value4")

	if cache.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cache.Size())
	}

	_, found := cache.Get("key1")
	if !found {
		t.Error("Expected key1 to exist (was accessed)")
	}

	_, found = cache.Get("key2")
	if found {
		t.Error("Expected key2 to be evicted (LRU)")
	}

	_, found = cache.Get("key3")
	if !found {
		t.Error("Expected key3 to exist")
	}

	_, found = cache.Get("key4")
	if !found {
		t.Error("Expected key4 to exist")
	}
}

func TestGlobalCache_SetCapacity(t *testing.T) {
	cache := NewGlobalCache()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	cache.SetCapacity(2)

	if cache.Size() != 2 {
		t.Errorf("Expected size 2 after setting capacity, got %d", cache.Size())
	}

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be evicted (LRU)")
	}

	_, found = cache.Get("key2")
	if !found {
		t.Error("Expected key2 to exist")
	}

	_, found = cache.Get("key3")
	if !found {
		t.Error("Expected key3 to exist")
	}
}

func TestGlobalCache_NoCapacityLimit(t *testing.T) {
	cache := NewGlobalCache()

	for i := 0; i < 1000; i++ {
		cache.Set(string(rune('a'+(i%26)))+string(rune('a'+((i/26)%26)))+string(rune('a'+((i/676)%26)))+string(rune('a'+((i/17576)%26))), i)
	}

	if cache.Size() != 1000 {
		t.Errorf("Expected size 1000, got %d", cache.Size())
	}
}

func TestGoroutineCache_SetAndGet(t *testing.T) {
	cache := NewGoroutineCache()

	cache.Set("key1", "value1")
	value, found := cache.Get("key1")

	if !found {
		t.Error("Expected to find key1")
	}

	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}
}

func TestGoroutineCache_SetWithExpiration(t *testing.T) {
	cache := NewGoroutineCache()

	cache.SetWithExpiration("key1", "value1", 100*time.Millisecond)
	value, found := cache.Get("key1")

	if !found {
		t.Error("Expected to find key1 immediately")
	}

	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	time.Sleep(150 * time.Millisecond)

	_, found = cache.Get("key1")
	if found {
		t.Error("Expected key1 to be expired")
	}
}

func TestGoroutineCache_Delete(t *testing.T) {
	cache := NewGoroutineCache()

	cache.Set("key1", "value1")
	cache.Delete("key1")

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be deleted")
	}
}

func TestGoroutineCache_Clear(t *testing.T) {
	cache := NewGoroutineCache()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0, got %d", cache.Size())
	}
}

func TestGoroutineCache_Size(t *testing.T) {
	cache := NewGoroutineCache()

	if cache.Size() != 0 {
		t.Errorf("Expected initial size 0, got %d", cache.Size())
	}

	cache.Set("key1", "value1")
	if cache.Size() != 1 {
		t.Errorf("Expected size 1, got %d", cache.Size())
	}

	cache.Set("key2", "value2")
	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}
}

func TestGoroutineCache_Keys(t *testing.T) {
	cache := NewGoroutineCache()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	keys := cache.Keys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	if !keyMap["key1"] || !keyMap["key2"] || !keyMap["key3"] {
		t.Error("Expected keys key1, key2, key3")
	}
}

func TestGoroutineCache_Isolation(t *testing.T) {
	cache1 := NewGoroutineCache()
	cache2 := NewGoroutineCache()

	cache1.Set("key1", "value1")
	cache2.Set("key1", "value2")

	value1, found1 := cache1.Get("key1")
	value2, found2 := cache2.Get("key1")

	if !found1 || value1 != "value1" {
		t.Error("Expected cache1 to have value1")
	}

	if !found2 || value2 != "value2" {
		t.Error("Expected cache2 to have value2")
	}
}

func TestGoroutineCache_CleanupExpired(t *testing.T) {
	cache := NewGoroutineCache()

	cache.Set("permanent", "value")
	cache.SetWithExpiration("expired1", "value1", 50*time.Millisecond)
	cache.SetWithExpiration("expired2", "value2", 50*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	cache.CleanupExpired()

	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1 after cleanup, got %d", cache.Size())
	}

	_, found := cache.Get("permanent")
	if !found {
		t.Error("Expected permanent key to still exist")
	}

	_, found = cache.Get("expired1")
	if found {
		t.Error("Expected expired1 to be cleaned up")
	}

	_, found = cache.Get("expired2")
	if found {
		t.Error("Expected expired2 to be cleaned up")
	}
}

func TestGoroutineCache_Capacity(t *testing.T) {
	cache := NewGoroutineCacheWithCapacity(3)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	if cache.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cache.Size())
	}

	cache.Set("key4", "value4")

	if cache.Size() != 3 {
		t.Errorf("Expected size 3 after adding 4th item, got %d", cache.Size())
	}

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be evicted (LRU)")
	}

	_, found = cache.Get("key2")
	if !found {
		t.Error("Expected key2 to exist")
	}

	_, found = cache.Get("key3")
	if !found {
		t.Error("Expected key3 to exist")
	}

	_, found = cache.Get("key4")
	if !found {
		t.Error("Expected key4 to exist")
	}
}

func TestGoroutineCache_LRU(t *testing.T) {
	cache := NewGoroutineCacheWithCapacity(3)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	cache.Get("key1")

	cache.Set("key4", "value4")

	if cache.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cache.Size())
	}

	_, found := cache.Get("key1")
	if !found {
		t.Error("Expected key1 to exist (was accessed)")
	}

	_, found = cache.Get("key2")
	if found {
		t.Error("Expected key2 to be evicted (LRU)")
	}

	_, found = cache.Get("key3")
	if !found {
		t.Error("Expected key3 to exist")
	}

	_, found = cache.Get("key4")
	if !found {
		t.Error("Expected key4 to exist")
	}
}

func TestGoroutineCache_SetCapacity(t *testing.T) {
	cache := NewGoroutineCache()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	cache.SetCapacity(2)

	if cache.Size() != 2 {
		t.Errorf("Expected size 2 after setting capacity, got %d", cache.Size())
	}

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be evicted (LRU)")
	}

	_, found = cache.Get("key2")
	if !found {
		t.Error("Expected key2 to exist")
	}

	_, found = cache.Get("key3")
	if !found {
		t.Error("Expected key3 to exist")
	}
}

func TestGoroutineCache_NoCapacityLimit(t *testing.T) {
	cache := NewGoroutineCache()

	for i := 0; i < 1000; i++ {
		cache.Set(string(rune('a'+(i%26)))+string(rune('a'+((i/26)%26)))+string(rune('a'+((i/676)%26)))+string(rune('a'+((i/17576)%26))), i)
	}

	if cache.Size() != 1000 {
		t.Errorf("Expected size 1000, got %d", cache.Size())
	}
}

func TestContextCache_SetAndGet(t *testing.T) {
	ctx := WithCache(context.Background())
	cache := NewContextCache(ctx)

	cache.Set("key1", "value1")
	value, found := cache.Get("key1")

	if !found {
		t.Error("Expected to find key1")
	}

	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}
}

func TestContextCache_SetWithExpiration(t *testing.T) {
	ctx := WithCache(context.Background())
	cache := NewContextCache(ctx)

	cache.SetWithExpiration("key1", "value1", 100*time.Millisecond)
	value, found := cache.Get("key1")

	if !found {
		t.Error("Expected to find key1 immediately")
	}

	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	time.Sleep(150 * time.Millisecond)

	_, found = cache.Get("key1")
	if found {
		t.Error("Expected key1 to be expired")
	}
}

func TestContextCache_Delete(t *testing.T) {
	ctx := WithCache(context.Background())
	cache := NewContextCache(ctx)

	cache.Set("key1", "value1")
	cache.Delete("key1")

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be deleted")
	}
}

func TestContextCache_Clear(t *testing.T) {
	ctx := WithCache(context.Background())
	cache := NewContextCache(ctx)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0, got %d", cache.Size())
	}
}

func TestContextCache_Size(t *testing.T) {
	ctx := WithCache(context.Background())
	cache := NewContextCache(ctx)

	if cache.Size() != 0 {
		t.Errorf("Expected initial size 0, got %d", cache.Size())
	}

	cache.Set("key1", "value1")
	if cache.Size() != 1 {
		t.Errorf("Expected size 1, got %d", cache.Size())
	}

	cache.Set("key2", "value2")
	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}
}

func TestContextCache_Keys(t *testing.T) {
	ctx := WithCache(context.Background())
	cache := NewContextCache(ctx)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	keys := cache.Keys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	if !keyMap["key1"] || !keyMap["key2"] || !keyMap["key3"] {
		t.Error("Expected keys key1, key2, key3")
	}
}

func TestContextCache_Isolation(t *testing.T) {
	ctx1 := WithCache(context.Background())
	ctx2 := WithCache(context.Background())

	cache1 := NewContextCache(ctx1)
	cache2 := NewContextCache(ctx2)

	cache1.Set("key1", "value1")
	cache2.Set("key1", "value2")

	value1, found1 := cache1.Get("key1")
	value2, found2 := cache2.Get("key1")

	if !found1 || value1 != "value1" {
		t.Error("Expected cache1 to have value1")
	}

	if !found2 || value2 != "value2" {
		t.Error("Expected cache2 to have value2")
	}
}

func TestContextCache_WithGoroutines(t *testing.T) {
	ctx := WithCache(context.Background())
	var wg sync.WaitGroup

	numGoroutines := 10
	numOperations := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			cache := NewContextCache(ctx)
			for j := 0; j < numOperations; j++ {
				key := string(rune('a' + (j % 26)))
				cache.Set(key, id)
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()

	cache := NewContextCache(ctx)
	if cache.Size() != 26 {
		t.Errorf("Expected cache size 26, got %d", cache.Size())
	}
}

func TestContextCache_GetCache(t *testing.T) {
	ctx := context.Background()

	cache := GetCache(ctx)
	if cache != nil {
		t.Error("Expected nil cache for context without cache")
	}

	ctx = WithCache(ctx)
	cache = GetCache(ctx)
	if cache == nil {
		t.Error("Expected non-nil cache for context with cache")
	}
}

func TestContextCache_NilContext(t *testing.T) {
	cache := NewContextCache(nil)

	cache.Set("key1", "value1")
	_, found := cache.Get("key1")

	if found {
		t.Error("Expected not to find key1 with nil context")
	}

	if cache.Size() != 0 {
		t.Errorf("Expected size 0 with nil context, got %d", cache.Size())
	}
}

func TestContextCache_Capacity(t *testing.T) {
	ctx := WithCacheCapacity(context.Background(), 3)
	cache := NewContextCache(ctx)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	if cache.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cache.Size())
	}

	cache.Set("key4", "value4")

	if cache.Size() != 3 {
		t.Errorf("Expected size 3 after adding 4th item, got %d", cache.Size())
	}

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be evicted (LRU)")
	}

	_, found = cache.Get("key2")
	if !found {
		t.Error("Expected key2 to exist")
	}

	_, found = cache.Get("key3")
	if !found {
		t.Error("Expected key3 to exist")
	}

	_, found = cache.Get("key4")
	if !found {
		t.Error("Expected key4 to exist")
	}
}

func TestContextCache_LRU(t *testing.T) {
	ctx := WithCacheCapacity(context.Background(), 3)
	cache := NewContextCache(ctx)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	cache.Get("key1")

	cache.Set("key4", "value4")

	if cache.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cache.Size())
	}

	_, found := cache.Get("key1")
	if !found {
		t.Error("Expected key1 to exist (was accessed)")
	}

	_, found = cache.Get("key2")
	if found {
		t.Error("Expected key2 to be evicted (LRU)")
	}

	_, found = cache.Get("key3")
	if !found {
		t.Error("Expected key3 to exist")
	}

	_, found = cache.Get("key4")
	if !found {
		t.Error("Expected key4 to exist")
	}
}

func TestContextCache_SetCapacity(t *testing.T) {
	ctx := WithCache(context.Background())
	cache := NewContextCache(ctx)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	cache.SetCapacity(2)

	if cache.Size() != 2 {
		t.Errorf("Expected size 2 after setting capacity, got %d", cache.Size())
	}

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be evicted (LRU)")
	}

	_, found = cache.Get("key2")
	if !found {
		t.Error("Expected key2 to exist")
	}

	_, found = cache.Get("key3")
	if !found {
		t.Error("Expected key3 to exist")
	}
}

func TestCacheItem_IsExpired(t *testing.T) {
	item := &CacheItem{
		Value:      "value",
		Expiration: 0,
	}

	if item.IsExpired() {
		t.Error("Expected item with zero expiration to not be expired")
	}

	item.Expiration = time.Now().Add(-1 * time.Hour).UnixNano()
	if !item.IsExpired() {
		t.Error("Expected item with past expiration to be expired")
	}

	item.Expiration = time.Now().Add(1 * time.Hour).UnixNano()
	if item.IsExpired() {
		t.Error("Expected item with future expiration to not be expired")
	}
}

func TestGlobalCache_StartAndStopCleanup(t *testing.T) {
	cache := NewGlobalCache()
	cache.SetWithExpiration("key1", "value1", 50*time.Millisecond)
	cache.Set("key2", "value2")

	cache.StartCleanup(30 * time.Millisecond)

	// 等待清理发生
	time.Sleep(150 * time.Millisecond)

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be cleaned up")
	}

	_, found = cache.Get("key2")
	if !found {
		t.Error("Expected key2 to still exist")
	}

	// 停止清理
	cache.StopCleanup()

	// 再次停止不应 panic
	cache.StopCleanup()
}

func TestGlobalCache_StartCleanupIdempotent(t *testing.T) {
	cache := NewGlobalCache()

	// 多次启动不应创建多个 goroutine
	cache.StartCleanup(100 * time.Millisecond)
	cache.StartCleanup(100 * time.Millisecond) // 应被忽略

	cache.StopCleanup()
}

func TestGoroutineCache_StartAndStopCleanup(t *testing.T) {
	cache := NewGoroutineCache()
	cache.SetWithExpiration("key1", "value1", 50*time.Millisecond)

	cache.StartCleanup(30 * time.Millisecond)
	time.Sleep(150 * time.Millisecond)

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be cleaned up")
	}

	cache.StopCleanup()
}
