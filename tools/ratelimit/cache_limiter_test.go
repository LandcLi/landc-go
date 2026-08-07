package ratelimit

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

// memoryCache 内存版 Cache（测试用）：并发安全 + 原子 Incr。
type memoryCache struct {
	mu sync.Mutex
	m  map[string]memoryEntry
}

type memoryEntry struct {
	value     string
	expiresAt time.Time
}

func newMemoryCache() *memoryCache {
	return &memoryCache{m: make(map[string]memoryEntry)}
}

func (c *memoryCache) Get(key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok || time.Now().After(e.expiresAt) {
		return "", nil
	}
	return e.value, nil
}

func (c *memoryCache) Set(key, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = memoryEntry{value: value, expiresAt: time.Now().Add(ttl)}
	return nil
}

func (c *memoryCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, key)
	return nil
}

func (c *memoryCache) Exists(key string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok || time.Now().After(e.expiresAt) {
		return false, nil
	}
	return true, nil
}

func (c *memoryCache) Incr(key string, ttl time.Duration) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.m[key]
	if time.Now().After(e.expiresAt) {
		e = memoryEntry{}
	}
	n, _ := strconv.ParseInt(e.value, 10, 64)
	n++
	c.m[key] = memoryEntry{value: strconv.FormatInt(n, 10), expiresAt: time.Now().Add(ttl)}
	return n, nil
}

func TestIntervalLimiter(t *testing.T) {
	c := newMemoryCache()
	l := NewIntervalLimiter(c)

	if !l.Allow("sms:13800000000", 60*time.Second) {
		t.Error("first allow should pass")
	}
	if l.Allow("sms:13800000000", 60*time.Second) {
		t.Error("second allow within window should be rejected")
	}
	// 不同 key 不受影响
	if !l.Allow("sms:13900000000", 60*time.Second) {
		t.Error("different key should pass")
	}
}

func TestIntervalLimiterExpiry(t *testing.T) {
	c := newMemoryCache()
	l := NewIntervalLimiter(c)
	_ = l.Allow("k", 50*time.Millisecond)
	if l.Allow("k", 50*time.Millisecond) {
		t.Error("within window should be rejected")
	}
	time.Sleep(60 * time.Millisecond)
	if !l.Allow("k", 50*time.Millisecond) {
		t.Error("after expiry should pass")
	}
}

func TestCountLimiter(t *testing.T) {
	c := newMemoryCache()
	l := NewCountLimiter(c)
	for i := 0; i < 3; i++ {
		if !l.Allow("send:daily:u1", 3, 24*time.Hour) {
			t.Fatalf("allow %d should pass", i)
		}
	}
	if l.Allow("send:daily:u1", 3, 24*time.Hour) {
		t.Error("4th allow should be rejected")
	}
}

func TestCountLimiterConcurrentIncr(t *testing.T) {
	c := newMemoryCache()
	l := NewCountLimiter(c)
	const goroutines, per = 8, 50
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < per; j++ {
				_ = l.Allow("k", 100000, time.Hour)
			}
		}()
	}
	wg.Wait()
	// 计数必须精确等于 goroutines*per（原子 Incr 不丢计数）
	v, _ := c.Get("k")
	if v != "400" {
		t.Errorf("counter = %s, want 400 (atomic Incr)", v)
	}
}
