package lock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================
// MemoryLock — 基于进程内共享状态的分布式锁实现
// 语义与 RedisLock 一致（token 校验 + TTL 过期 + 自动续约），
// 适用于单机部署与单元测试；多实例场景请使用 RedisLock。
// ============================================================

type memLockEntry struct {
	value    string
	expireAt time.Time
}

var (
	memRegistryMu sync.Mutex
	memLocks      = make(map[string]*memLockEntry)
)

type MemoryLock struct {
	name      string
	value     string
	config    *LockConfig
	renewStop chan struct{}
	mu        sync.Mutex
	locked    bool
}

// NewMemoryLock 创建内存锁
func NewMemoryLock(name string, opts ...LockOption) *MemoryLock {
	cfg := DefaultLockConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return &MemoryLock{
		name:      name,
		value:     uuid.New().String(),
		config:    cfg,
		renewStop: make(chan struct{}),
	}
}

// MemoryLockFactory 基于内存实现的锁工厂
type MemoryLockFactory struct{}

// NewMemoryLockFactory 创建内存锁工厂
func NewMemoryLockFactory() *MemoryLockFactory { return &MemoryLockFactory{} }

func (f *MemoryLockFactory) NewLock(name string, opts ...LockOption) DistributedLock {
	return NewMemoryLock(name, opts...)
}

func (l *MemoryLock) Lock(ctx context.Context) error {
	for i := 0; i < l.config.Retry; i++ {
		ok, err := l.tryLock(ctx)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(l.config.RetryWait):
		}
	}
	return fmt.Errorf("lock: failed to acquire lock after %d retries: %s", l.config.Retry, l.name)
}

func (l *MemoryLock) TryLock(ctx context.Context) (bool, error) {
	return l.tryLock(ctx)
}

func (l *MemoryLock) tryLock(ctx context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	memRegistryMu.Lock()
	defer memRegistryMu.Unlock()

	now := time.Now()
	if entry, ok := memLocks[l.name]; ok && entry.expireAt.After(now) {
		return false, nil
	}
	memLocks[l.name] = &memLockEntry{
		value:    l.value,
		expireAt: now.Add(l.config.TTL),
	}
	l.locked = true
	if l.config.AutoRenew {
		l.startAutoRenew(ctx)
	}
	return true, nil
}

func (l *MemoryLock) Unlock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.locked {
		return nil
	}
	defer func() { l.locked = false }()

	if l.config.AutoRenew {
		close(l.renewStop)
	}

	memRegistryMu.Lock()
	defer memRegistryMu.Unlock()

	// 仅当持有者 token 匹配时才释放，防止误删其他持有者的锁
	if entry, ok := memLocks[l.name]; ok && entry.value == l.value {
		delete(memLocks, l.name)
	}
	return nil
}

func (l *MemoryLock) startAutoRenew(ctx context.Context) {
	ticker := time.NewTicker(l.config.TTL / 3)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				memRegistryMu.Lock()
				if entry, ok := memLocks[l.name]; ok && entry.value == l.value {
					entry.expireAt = time.Now().Add(l.config.TTL)
				}
				memRegistryMu.Unlock()
			case <-l.renewStop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}
