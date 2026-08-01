package idempotent

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestMemoryIdempotencyChecker 验证内存幂等检查器基本流程
func TestMemoryIdempotencyChecker(t *testing.T) {
	c := NewMemoryIdempotencyChecker(time.Hour)
	ctx := context.Background()

	// 未处理 → 处理 → 已处理
	key := c.GenerateKey("wf-1", "exec-1")
	if processed, err := c.IsProcessed(ctx, key); err != nil || processed {
		t.Fatalf("expected not processed, got processed=%v err=%v", processed, err)
	}
	if err := c.MarkProcessed(ctx, key, 0); err != nil {
		t.Fatalf("mark processed: %v", err)
	}
	if processed, err := c.IsProcessed(ctx, key); err != nil || !processed {
		t.Fatalf("expected processed, got processed=%v err=%v", processed, err)
	}
}

// TestMemoryIdempotencyCheckerExpiry 验证过期后失效
func TestMemoryIdempotencyCheckerExpiry(t *testing.T) {
	c := NewMemoryIdempotencyChecker(0) // 默认 24h，但这里用短 ttl 覆盖
	ctx := context.Background()

	key := "expiring-key"
	if err := c.MarkProcessed(ctx, key, 50*time.Millisecond); err != nil {
		t.Fatalf("mark processed: %v", err)
	}
	if processed, _ := c.IsProcessed(ctx, key); !processed {
		t.Fatal("key should be processed before expiry")
	}
	time.Sleep(80 * time.Millisecond)
	if processed, _ := c.IsProcessed(ctx, key); processed {
		t.Fatal("key should have expired")
	}
}

// TestMemoryIdempotencyCheckerConcurrent 验证并发安全（-race 检测）
func TestMemoryIdempotencyCheckerConcurrent(t *testing.T) {
	c := NewMemoryIdempotencyChecker(time.Hour)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := c.GenerateKey("wf", "exec", string(rune('a'+n%26)))
			_ = c.MarkProcessed(ctx, key, 0)
			_, _ = c.IsProcessed(ctx, key)
		}(i)
	}
	wg.Wait()
}

// TestGenerateKeyDeterministic 验证相同输入生成相同 key，不同输入生成不同 key
func TestGenerateKeyDeterministic(t *testing.T) {
	c := NewMemoryIdempotencyChecker(time.Hour)
	k1 := c.GenerateKey("a", "b", "c")
	k2 := c.GenerateKey("a", "b", "c")
	if k1 != k2 {
		t.Errorf("same input should produce same key: %s vs %s", k1, k2)
	}
	k3 := c.GenerateKey("a", "b", "d")
	if k1 == k3 {
		t.Errorf("different input should produce different key")
	}
}
