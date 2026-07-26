package lock

import (
	"context"
	"time"
)

// DistributedLock 分布式锁接口
type DistributedLock interface {
	// Lock 获取锁，阻塞直到获取成功或上下文取消
	Lock(ctx context.Context) error
	// TryLock 尝试获取锁，立即返回结果
	TryLock(ctx context.Context) (bool, error)
	// Unlock 释放锁
	Unlock(ctx context.Context) error
}

// LockFactory 分布式锁工厂
type LockFactory interface {
	// NewLock 创建一个新的分布式锁
	NewLock(name string, opts ...LockOption) DistributedLock
}

// LockOption 锁配置选项
type LockOption func(*LockConfig)

// LockConfig 锁配置
type LockConfig struct {
	TTL       time.Duration // 锁持有时间（自动续约时用于延长）
	Retry     int           // 尝试获取锁的重试次数
	RetryWait time.Duration // 重试等待时间
	AutoRenew bool          // 是否自动续约
}

// DefaultLockConfig 默认锁配置
func DefaultLockConfig() *LockConfig {
	return &LockConfig{
		TTL:       30 * time.Second,
		Retry:     3,
		RetryWait: 100 * time.Millisecond,
		AutoRenew: false,
	}
}

// WithTTL 设置锁 TTL
func WithTTL(ttl time.Duration) LockOption {
	return func(c *LockConfig) {
		c.TTL = ttl
	}
}

// WithRetry 设置重试次数
func WithRetry(retry int) LockOption {
	return func(c *LockConfig) {
		c.Retry = retry
	}
}

// WithRetryWait 设置重试等待时间
func WithRetryWait(wait time.Duration) LockOption {
	return func(c *LockConfig) {
		c.RetryWait = wait
	}
}

// WithAutoRenew 设置自动续约
func WithAutoRenew(renew bool) LockOption {
	return func(c *LockConfig) {
		c.AutoRenew = renew
	}
}
