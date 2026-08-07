package verifycode

import (
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/tools/ratelimit"
)

// memoryStore 内存版 ratelimit.Cache（测试用，并发安全 + 原子 Incr）。
type memoryStore struct {
	mu sync.Mutex
	m  map[string]memoryEntry
}

type memoryEntry struct {
	value     string
	expiresAt time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{m: make(map[string]memoryEntry)}
}

func (c *memoryStore) Get(key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok || time.Now().After(e.expiresAt) {
		return "", nil
	}
	return e.value, nil
}

func (c *memoryStore) Set(key, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = memoryEntry{value: value, expiresAt: time.Now().Add(ttl)}
	return nil
}

func (c *memoryStore) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, key)
	return nil
}

func (c *memoryStore) Exists(key string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok || time.Now().After(e.expiresAt) {
		return false, nil
	}
	return true, nil
}

func (c *memoryStore) Incr(key string, ttl time.Duration) (int64, error) {
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

var _ ratelimit.Cache = (*memoryStore)(nil)

func TestGenerateAndVerify(t *testing.T) {
	m := NewManager(newMemoryStore())

	code, err := m.Generate("sms:13800000000")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("code length = %d, want 6", len(code))
	}
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			t.Fatalf("code %q contains non-digit", code)
		}
	}

	if !m.Verify("sms:13800000000", code) {
		t.Error("Verify should pass with correct code")
	}
	// 一次性：第二次校验失败（已删除）
	if m.Verify("sms:13800000000", code) {
		t.Error("Verify should fail after one-time consumption")
	}
}

func TestVerifyWrongCode(t *testing.T) {
	m := NewManager(newMemoryStore())
	_, _ = m.Generate("k")
	if m.Verify("k", "000000") {
		t.Error("wrong code should fail")
	}
}

func TestSendIntervalLimited(t *testing.T) {
	m := NewManager(newMemoryStore())
	_, err := m.Generate("k")
	if err != nil {
		t.Fatal(err)
	}
	// 立即再次生成 → 间隔未到
	if _, err := m.Generate("k"); !errors.Is(err, ErrIntervalLimited) {
		t.Fatalf("second Generate = %v, want ErrIntervalLimited", err)
	}
	// 不同 key 不受影响
	if _, err := m.Generate("other"); err != nil {
		t.Fatalf("different key Generate = %v, want ok", err)
	}
}

func TestDailyLimit(t *testing.T) {
	m := NewManager(newMemoryStore(), WithDailyLimit(3), WithSendInterval(time.Nanosecond))
	limited := false
	for i := 0; i < 5; i++ {
		_, err := m.Generate("k")
		if errors.Is(err, ErrDailyLimited) {
			limited = true
			break
		}
	}
	if !limited {
		t.Error("should hit daily limit after 3 sends")
	}
}

func TestCodeTTL(t *testing.T) {
	m := NewManager(newMemoryStore(), WithTTL(50*time.Millisecond))
	code, _ := m.Generate("k")
	time.Sleep(60 * time.Millisecond)
	if m.Verify("k", code) {
		t.Error("code should expire after TTL")
	}
}
