package cache

import (
	"container/list"
	"context"
	"sync"
	"time"
)

// Cache 缓存接口
type Cache interface {
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
	Delete(key string)
	Clear()
	Size() int
	Keys() []string
}

// CacheItem 缓存条目
type CacheItem struct {
	Value      interface{}
	Expiration int64
	accessTime int64
}

// IsExpired 检查条目是否已过期
func (item *CacheItem) IsExpired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

type lruEntry struct {
	key   string
	value *CacheItem
}

// lruCache 基础 LRU 缓存实现（内部使用，消除 GlobalCache/GoroutineCache 重复代码）
type lruCache struct {
	items    map[string]*list.Element
	lruList  *list.List
	capacity int
	mu       sync.RWMutex

	// cleanup 控制
	stopChan chan struct{}
	stopped  bool
}

func newLRUCache(capacity int) *lruCache {
	return &lruCache{
		items:    make(map[string]*list.Element),
		lruList:  list.New(),
		capacity: capacity,
	}
}

func (c *lruCache) Set(key string, value interface{}) {
	c.SetWithExpiration(key, value, 0)
}

func (c *lruCache) SetWithExpiration(key string, value interface{}, expiration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var exp int64
	if expiration > 0 {
		exp = time.Now().Add(expiration).UnixNano()
	}

	item := &CacheItem{
		Value:      value,
		Expiration: exp,
		accessTime: time.Now().UnixNano(),
	}

	if elem, exists := c.items[key]; exists {
		entry := elem.Value.(*lruEntry)
		entry.value = item
		c.lruList.MoveToFront(elem)
		return
	}

	elem := c.lruList.PushFront(&lruEntry{key: key, value: item})
	c.items[key] = elem

	if c.capacity > 0 && c.lruList.Len() > c.capacity {
		c.evict()
	}
}

func (c *lruCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, found := c.items[key]
	if !found {
		return nil, false
	}

	item := elem.Value.(*lruEntry).value

	if item.IsExpired() {
		c.removeElement(elem)
		return nil, false
	}

	item.accessTime = time.Now().UnixNano()
	c.lruList.MoveToFront(elem)

	return item.Value, true
}

func (c *lruCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.items[key]; exists {
		c.removeElement(elem)
	}
}

func (c *lruCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.lruList.Init()
}

func (c *lruCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lruList.Len()
}

func (c *lruCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, c.lruList.Len())
	for elem := c.lruList.Front(); elem != nil; elem = elem.Next() {
		keys = append(keys, elem.Value.(*lruEntry).key)
	}
	return keys
}

func (c *lruCache) CleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixNano()
	for elem := c.lruList.Back(); elem != nil; {
		item := elem.Value.(*lruEntry).value
		if item.Expiration > 0 && now > item.Expiration {
			next := elem.Prev()
			c.removeElement(elem)
			elem = next
		} else {
			elem = elem.Prev()
		}
	}
}

// StartCleanup 启动定时清理过期条目的后台 goroutine
func (c *lruCache) StartCleanup(interval time.Duration) {
	c.mu.Lock()
	if c.stopChan != nil && !c.stopped {
		c.mu.Unlock()
		return // 已经在运行
	}
	c.stopChan = make(chan struct{})
	c.stopped = false
	c.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.CleanupExpired()
			case <-c.stopChan:
				return
			}
		}
	}()
}

// StopCleanup 停止定时清理
func (c *lruCache) StopCleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopChan != nil && !c.stopped {
		close(c.stopChan)
		c.stopped = true
	}
}

func (c *lruCache) evict() {
	if c.lruList.Len() == 0 {
		return
	}

	elem := c.lruList.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

func (c *lruCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*lruEntry)
	delete(c.items, entry.key)
	c.lruList.Remove(elem)
}

func (c *lruCache) Capacity() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.capacity
}

func (c *lruCache) SetCapacity(capacity int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.capacity = capacity
	for c.capacity > 0 && c.lruList.Len() > c.capacity {
		c.evict()
	}
}

// ===================== GlobalCache =====================

// GlobalCache 全局缓存（进程级别共享）
type GlobalCache struct {
	*lruCache
}

// NewGlobalCache 创建全局缓存
func NewGlobalCache() *GlobalCache {
	return &GlobalCache{lruCache: newLRUCache(0)}
}

// NewGlobalCacheWithCapacity 创建带容量限制的全局缓存
func NewGlobalCacheWithCapacity(capacity int) *GlobalCache {
	return &GlobalCache{lruCache: newLRUCache(capacity)}
}

// ===================== GoroutineCache =====================

// GoroutineCache 协程级缓存
type GoroutineCache struct {
	*lruCache
}

// NewGoroutineCache 创建协程级缓存
func NewGoroutineCache() *GoroutineCache {
	return &GoroutineCache{lruCache: newLRUCache(0)}
}

// NewGoroutineCacheWithCapacity 创建带容量限制的协程级缓存
func NewGoroutineCacheWithCapacity(capacity int) *GoroutineCache {
	return &GoroutineCache{lruCache: newLRUCache(capacity)}
}

// ===================== ContextCache =====================

// ContextCache 基于 context 的缓存
type ContextCache struct {
	ctx context.Context
}

// NewContextCache 创建基于 context 的缓存
func NewContextCache(ctx context.Context) *ContextCache {
	return &ContextCache{ctx: ctx}
}

func (c *ContextCache) Set(key string, value interface{}) {
	c.SetWithExpiration(key, value, 0)
}

func (c *ContextCache) SetWithExpiration(key string, value interface{}, expiration time.Duration) {
	cache := c.getCache()
	if cache == nil {
		return
	}
	cache.SetWithExpiration(key, value, expiration)
}

func (c *ContextCache) Get(key string) (interface{}, bool) {
	cache := c.getCache()
	if cache == nil {
		return nil, false
	}
	return cache.Get(key)
}

func (c *ContextCache) Delete(key string) {
	cache := c.getCache()
	if cache == nil {
		return
	}
	cache.Delete(key)
}

func (c *ContextCache) Clear() {
	cache := c.getCache()
	if cache == nil {
		return
	}
	cache.Clear()
}

func (c *ContextCache) Size() int {
	cache := c.getCache()
	if cache == nil {
		return 0
	}
	return cache.Size()
}

func (c *ContextCache) Keys() []string {
	cache := c.getCache()
	if cache == nil {
		return nil
	}
	return cache.Keys()
}

func (c *ContextCache) CleanupExpired() {
	cache := c.getCache()
	if cache == nil {
		return
	}
	cache.CleanupExpired()
}

func (c *ContextCache) Capacity() int {
	cache := c.getCache()
	if cache == nil {
		return 0
	}
	return cache.Capacity()
}

func (c *ContextCache) SetCapacity(capacity int) {
	cache := c.getCache()
	if cache == nil {
		return
	}
	cache.SetCapacity(capacity)
}

func (c *ContextCache) getCache() *GoroutineCache {
	if c.ctx == nil {
		return nil
	}
	cache, ok := c.ctx.Value(cacheKey{}).(*GoroutineCache)
	if !ok {
		return nil
	}
	return cache
}

type cacheKey struct{}

// WithCache 向 context 中注入缓存
func WithCache(ctx context.Context) context.Context {
	cache := NewGoroutineCache()
	return context.WithValue(ctx, cacheKey{}, cache)
}

// WithCacheCapacity 向 context 中注入带容量限制的缓存
func WithCacheCapacity(ctx context.Context, capacity int) context.Context {
	cache := NewGoroutineCacheWithCapacity(capacity)
	return context.WithValue(ctx, cacheKey{}, cache)
}

// GetCache 从 context 中获取缓存
func GetCache(ctx context.Context) *GoroutineCache {
	if ctx == nil {
		return nil
	}
	cache, ok := ctx.Value(cacheKey{}).(*GoroutineCache)
	if !ok {
		return nil
	}
	return cache
}
