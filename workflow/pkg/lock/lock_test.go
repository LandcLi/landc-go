package lock

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestDefaultLockConfig 验证默认锁配置
func TestDefaultLockConfig(t *testing.T) {
	cfg := DefaultLockConfig()
	if cfg.TTL != 30*time.Second {
		t.Errorf("default TTL = %v, want 30s", cfg.TTL)
	}
	if cfg.Retry != 3 {
		t.Errorf("default Retry = %d, want 3", cfg.Retry)
	}
	if cfg.RetryWait != 100*time.Millisecond {
		t.Errorf("default RetryWait = %v, want 100ms", cfg.RetryWait)
	}
	if cfg.AutoRenew {
		t.Error("default AutoRenew should be false")
	}
}

// TestLockOptions 验证各锁配置选项
func TestLockOptions(t *testing.T) {
	cfg := DefaultLockConfig()
	WithTTL(5 * time.Second)(cfg)
	WithRetry(7)(cfg)
	WithRetryWait(200 * time.Millisecond)(cfg)
	WithAutoRenew(true)(cfg)

	if cfg.TTL != 5*time.Second || cfg.Retry != 7 ||
		cfg.RetryWait != 200*time.Millisecond || !cfg.AutoRenew {
		t.Errorf("options not applied: %+v", cfg)
	}
}

// TestMemoryLockTryLockExclusive 验证互斥：同一锁名只能被一个实例持有
func TestMemoryLockTryLockExclusive(t *testing.T) {
	ctx := context.Background()
	l1 := NewMemoryLock("exclusive")
	l2 := NewMemoryLock("exclusive")

	ok1, err := l1.TryLock(ctx)
	if err != nil || !ok1 {
		t.Fatalf("first TryLock = %v, err=%v", ok1, err)
	}
	ok2, err := l2.TryLock(ctx)
	if err != nil {
		t.Fatalf("second TryLock err = %v", err)
	}
	if ok2 {
		t.Fatal("second TryLock should fail while first holds the lock")
	}

	if err := l1.Unlock(ctx); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	ok3, err := l2.TryLock(ctx)
	if err != nil || !ok3 {
		t.Fatalf("TryLock after release = %v, err=%v", ok3, err)
	}
	_ = l2.Unlock(ctx)
}

// TestMemoryLockLockWithRetry 验证阻塞式 Lock 在释放后可获取
func TestMemoryLockLockWithRetry(t *testing.T) {
	l1 := NewMemoryLock("retry-lock", WithRetry(5), WithRetryWait(10*time.Millisecond))
	l2 := NewMemoryLock("retry-lock", WithRetry(5), WithRetryWait(10*time.Millisecond))

	ctx := context.Background()
	if ok, _ := l1.TryLock(ctx); !ok {
		t.Fatal("l1 should acquire")
	}

	var acquired bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := l2.Lock(ctx); err != nil {
			return
		}
		acquired = true
	}()

	// 稍后释放，l2 的重试应能拿到锁
	time.Sleep(30 * time.Millisecond)
	_ = l1.Unlock(ctx)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("l2 Lock did not return in time")
	}
	if !acquired {
		t.Fatal("l2 should acquire after l1 releases")
	}
}

// TestMemoryLockTTLExpiry 验证 TTL 过期后锁自动释放
func TestMemoryLockTTLExpiry(t *testing.T) {
	ctx := context.Background()
	l1 := NewMemoryLock("ttl-lock", WithTTL(50*time.Millisecond))
	l2 := NewMemoryLock("ttl-lock", WithTTL(50*time.Millisecond))

	if ok, _ := l1.TryLock(ctx); !ok {
		t.Fatal("l1 should acquire")
	}

	// 未过期：l2 拿不到
	if ok, _ := l2.TryLock(ctx); ok {
		t.Fatal("l2 should not acquire before TTL expiry")
	}

	// 等待 TTL 过期
	time.Sleep(120 * time.Millisecond)
	ok, err := l2.TryLock(ctx)
	if err != nil || !ok {
		t.Fatalf("l2 should acquire after TTL expiry: ok=%v err=%v", ok, err)
	}
	_ = l2.Unlock(ctx)
}

// TestMemoryLockAutoRenew 验证自动续约防止锁过期
func TestMemoryLockAutoRenew(t *testing.T) {
	ctx := context.Background()
	l1 := NewMemoryLock("renew-lock", WithTTL(100*time.Millisecond), WithAutoRenew(true))
	l2 := NewMemoryLock("renew-lock", WithTTL(100*time.Millisecond), WithAutoRenew(true))

	if ok, _ := l1.TryLock(ctx); !ok {
		t.Fatal("l1 should acquire")
	}

	// 等待超过一个 TTL，验证续约生效（锁仍在 l1 手中）
	time.Sleep(250 * time.Millisecond)
	ok, err := l2.TryLock(ctx)
	if err != nil {
		t.Fatalf("l2 TryLock err = %v", err)
	}
	if ok {
		t.Fatal("l2 should not acquire while l1 keeps renewing")
	}

	if err := l1.Unlock(ctx); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	// 停止续约后锁应最终过期
	time.Sleep(150 * time.Millisecond)
	if ok, _ := l2.TryLock(ctx); !ok {
		t.Fatal("l2 should acquire after l1 unlock")
	}
	_ = l2.Unlock(ctx)
}

// TestMemoryLockContextCancel 验证 Lock 在 ctx 取消时返回错误
func TestMemoryLockContextCancel(t *testing.T) {
	l1 := NewMemoryLock("cancel-lock", WithTTL(30*time.Second))
	l2 := NewMemoryLock("cancel-lock", WithTTL(30*time.Second))

	ctx := context.Background()
	if ok, _ := l1.TryLock(ctx); !ok {
		t.Fatal("l1 should acquire")
	}

	cctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := l2.Lock(cctx)
	if err == nil {
		t.Fatal("Lock should return error on context cancel")
	}
}

// TestMemoryLockUnlockIdempotent 验证重复释放/未持锁释放安全
func TestMemoryLockUnlockIdempotent(t *testing.T) {
	ctx := context.Background()
	l := NewMemoryLock("idempotent-lock")
	if err := l.Unlock(ctx); err != nil {
		t.Fatalf("unlock without holding should be nil, got %v", err)
	}
	if ok, _ := l.TryLock(ctx); !ok {
		t.Fatal("should acquire")
	}
	if err := l.Unlock(ctx); err != nil {
		t.Fatalf("first unlock: %v", err)
	}
	if err := l.Unlock(ctx); err != nil {
		t.Fatalf("second unlock should be nil, got %v", err)
	}
}

// TestMemoryLockFactory 验证内存锁工厂实现 LockFactory 接口
func TestMemoryLockFactory(t *testing.T) {
	f := NewMemoryLockFactory()
	_ = f.NewLock("factory-lock", WithTTL(time.Second))
	// 接口契约编译期验证
	var _ LockFactory = f
}

// TestMemoryLockConcurrent 并发抢锁：临界区计数必须严格等于并发数
func TestMemoryLockConcurrent(t *testing.T) {
	const goroutines = 20
	ctx := context.Background()

	var counter int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l := NewMemoryLock("concurrent-lock", WithTTL(200*time.Millisecond), WithRetry(50), WithRetryWait(5*time.Millisecond))
			if err := l.Lock(ctx); err != nil {
				t.Errorf("Lock failed: %v", err)
				return
			}
			mu.Lock()
			counter++
			mu.Unlock()
			time.Sleep(time.Millisecond)
			if err := l.Unlock(ctx); err != nil {
				t.Errorf("Unlock failed: %v", err)
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if counter != goroutines {
		t.Fatalf("counter = %d, want %d", counter, goroutines)
	}
}
