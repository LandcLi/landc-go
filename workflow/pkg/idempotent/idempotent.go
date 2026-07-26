package idempotent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// ============================================================
// IdempotencyChecker — 幂等性检查器
// 复用：底层存储使用 frame/pkg/cache.Cache（Redis / 本地 LRU）
// ============================================================

// IdempotencyStore 幂等存储接口
type IdempotencyStore interface {
	IsAttemptProcessed(ctx context.Context, attemptID string) (bool, error)
	MarkAttemptProcessed(ctx context.Context, attemptID string, ttl time.Duration) error
}

type IdempotencyChecker interface {
	IsProcessed(ctx context.Context, idempotencyKey string) (bool, error)
	MarkProcessed(ctx context.Context, idempotencyKey string, ttl time.Duration) error
	GenerateKey(parts ...string) string
}

// ============================================================
// StoreIdempotencyChecker — 基于 IdempotencyStore 的实现
// 不额外加前缀，由 store 自己管理命名空间
// ============================================================

type StoreIdempotencyChecker struct {
	store IdempotencyStore
	ttl   time.Duration
}

func NewStoreIdempotencyChecker(store IdempotencyStore, ttl time.Duration) *StoreIdempotencyChecker {
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return &StoreIdempotencyChecker{store: store, ttl: ttl}
}

func (c *StoreIdempotencyChecker) IsProcessed(ctx context.Context, idempotencyKey string) (bool, error) {
	return c.store.IsAttemptProcessed(ctx, idempotencyKey)
}

func (c *StoreIdempotencyChecker) MarkProcessed(ctx context.Context, idempotencyKey string, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.ttl
	}
	return c.store.MarkAttemptProcessed(ctx, idempotencyKey, ttl)
}

func (c *StoreIdempotencyChecker) GenerateKey(parts ...string) string {
	raw := ""
	for i, p := range parts {
		if i > 0 {
			raw += ":"
		}
		raw += p
	}
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

// ============================================================
// MemoryIdempotencyChecker — 开发/测试用
// ============================================================

type MemoryIdempotencyChecker struct {
	store map[string]time.Time
	ttl   time.Duration
}

func NewMemoryIdempotencyChecker(ttl time.Duration) *MemoryIdempotencyChecker {
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return &MemoryIdempotencyChecker{
		store: make(map[string]time.Time),
		ttl:   ttl,
	}
}

func (c *MemoryIdempotencyChecker) IsProcessed(_ context.Context, idempotencyKey string) (bool, error) {
	expires, ok := c.store[idempotencyKey]
	if !ok || time.Now().After(expires) {
		delete(c.store, idempotencyKey)
		return false, nil
	}
	return true, nil
}

func (c *MemoryIdempotencyChecker) MarkProcessed(_ context.Context, idempotencyKey string, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.ttl
	}
	c.store[idempotencyKey] = time.Now().Add(ttl)
	return nil
}

func (c *MemoryIdempotencyChecker) GenerateKey(parts ...string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(join(parts...))))
}

func join(parts ...string) string {
	result := parts[0]
	for _, p := range parts[1:] {
		result += ":" + p
	}
	return result
}
