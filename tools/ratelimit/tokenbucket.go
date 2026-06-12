// Package ratelimit 提供令牌桶限流算法（无任何 Web 框架依赖）。
//
// 典型用法（配合 Gin 的 limiter）:
//
//	tb := ratelimit.NewTokenBucket(100, 200) // 每秒 100 个，突发 200
//	if !tb.Allow() {
//	    c.AbortWithStatusJSON(429, gin.H{"error": "too many requests"})
//	    return
//	}
package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket 令牌桶限流器（线程安全）。
// 纯算法实现，不依赖任何 Web 框架。
type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // 每秒补充的令牌数
	lastRefill time.Time
}

// NewTokenBucket 创建令牌桶。
// rate 是每秒补充的令牌数，burst 是桶容量（最大突发请求数）。
func NewTokenBucket(rate float64, burst int) *TokenBucket {
	return &TokenBucket{
		tokens:     float64(burst),
		maxTokens:  float64(burst),
		refillRate: rate,
		lastRefill: time.Now(),
	}
}

// Allow 消费一个令牌，返回 true 表示允许通过。
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN 消费 n 个令牌，返回 true 表示允许通过。
func (tb *TokenBucket) AllowN(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.lastRefill = now

	// 补充令牌
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}

	// 消耗令牌
	if tb.tokens < float64(n) {
		return false
	}
	tb.tokens -= float64(n)
	return true
}

// Available 返回当前可用令牌数（近似值，用于监控）。
func (tb *TokenBucket) Available() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tokens := tb.tokens + elapsed*tb.refillRate
	if tokens > tb.maxTokens {
		tokens = tb.maxTokens
	}
	return tokens
}
