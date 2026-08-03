package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiterConfig 限流器配置
type RateLimiterConfig struct {
	// Rate 每秒允许的请求数
	Rate float64
	// Burst 突发容量（令牌桶大小）
	Burst int
	// KeyFunc 提取限流 key 的函数（默认按 ClientIP）
	KeyFunc func(c *gin.Context) string
	// ExceedHandler 超限处理函数（默认返回 429）
	ExceedHandler gin.HandlerFunc
}

// tokenBucket 令牌桶
type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

func newTokenBucket(rate float64, burst int) *tokenBucket {
	return &tokenBucket{
		tokens:     float64(burst),
		maxTokens:  float64(burst),
		refillRate: rate,
		lastRefill: time.Now(),
	}
}

func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// rateLimiterStore 存储每个 key 的令牌桶
type rateLimiterStore struct {
	buckets map[string]*tokenBucket
	mu      sync.RWMutex
	rate    float64
	burst   int
}

func newRateLimiterStore(rate float64, burst int) *rateLimiterStore {
	store := &rateLimiterStore{
		buckets: make(map[string]*tokenBucket),
		rate:    rate,
		burst:   burst,
	}
	// 后台清理过期桶
	go store.cleanup()
	return store
}

func (s *rateLimiterStore) getBucket(key string) *tokenBucket {
	s.mu.RLock()
	bucket, ok := s.buckets[key]
	s.mu.RUnlock()
	if ok {
		return bucket
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// 双重检查
	if bucket, ok := s.buckets[key]; ok {
		return bucket
	}
	bucket = newTokenBucket(s.rate, s.burst)
	s.buckets[key] = bucket
	return bucket
}

func (s *rateLimiterStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, bucket := range s.buckets {
			bucket.mu.Lock()
			// 超过 10 分钟没有请求的桶清理掉
			if now.Sub(bucket.lastRefill) > 10*time.Minute {
				delete(s.buckets, key)
			}
			bucket.mu.Unlock()
		}
		s.mu.Unlock()
	}
}

// RateLimiter 创建限流中间件
func RateLimiter(config RateLimiterConfig) gin.HandlerFunc {
	if config.Rate <= 0 {
		config.Rate = 100
	}
	if config.Burst <= 0 {
		config.Burst = int(config.Rate)
	}
	if config.KeyFunc == nil {
		config.KeyFunc = func(c *gin.Context) string {
			return c.ClientIP()
		}
	}

	store := newRateLimiterStore(config.Rate, config.Burst)

	return func(c *gin.Context) {
		key := config.KeyFunc(c)
		bucket := store.getBucket(key)

		if !bucket.allow() {
			if config.ExceedHandler != nil {
				config.ExceedHandler(c)
			} else {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"code":    42900,
					"message": "Too Many Requests",
				})
			}
			return
		}

		c.Next()
	}
}

// GlobalRateLimiter 全局限流（不区分 key，所有请求共享一个桶）
func GlobalRateLimiter(rate float64, burst int) gin.HandlerFunc {
	if rate <= 0 {
		rate = 1000
	}
	if burst <= 0 {
		burst = int(rate)
	}
	bucket := newTokenBucket(rate, burst)

	return func(c *gin.Context) {
		if !bucket.allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    42900,
				"message": "Too Many Requests",
			})
			return
		}
		c.Next()
	}
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	// Threshold 错误阈值（窗口内达到此数即熔断）
	Threshold int
	// Window 滑动窗口时间
	Window time.Duration
	// CoolDown 熔断后的冷却时间
	CoolDown time.Duration
	// IsFailure 判断是否为失败请求（默认 status >= 500）
	IsFailure func(c *gin.Context) bool
	// OnOpen 熔断打开时的回调
	OnOpen func(c *gin.Context)
}

// circuitBreakerState 熔断器状态
type circuitBreakerState int

const (
	stateClosed   circuitBreakerState = iota // 正常
	stateOpen                                // 熔断
	stateHalfOpen                            // 半开（试探）
)

type circuitBreaker struct {
	config   CircuitBreakerConfig
	state    circuitBreakerState
	failures []time.Time
	lastOpen time.Time
	mu       sync.Mutex
}

func newCircuitBreaker(config CircuitBreakerConfig) *circuitBreaker {
	return &circuitBreaker{
		config:   config,
		state:    stateClosed,
		failures: make([]time.Time, 0),
	}
}

func (cb *circuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case stateClosed:
		return true
	case stateOpen:
		if time.Since(cb.lastOpen) > cb.config.CoolDown {
			cb.state = stateHalfOpen
			return true
		}
		return false
	case stateHalfOpen:
		return true
	}
	return true
}

func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == stateHalfOpen {
		cb.state = stateClosed
		cb.failures = cb.failures[:0]
	}
}

func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	cb.failures = append(cb.failures, now)

	// 清理窗口外的记录
	windowStart := now.Add(-cb.config.Window)
	valid := cb.failures[:0]
	for _, t := range cb.failures {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}
	cb.failures = valid

	if cb.state == stateHalfOpen {
		cb.state = stateOpen
		cb.lastOpen = now
		return
	}

	if len(cb.failures) >= cb.config.Threshold {
		cb.state = stateOpen
		cb.lastOpen = now
	}
}

// CircuitBreaker 创建熔断器中间件
func CircuitBreaker(config CircuitBreakerConfig) gin.HandlerFunc {
	if config.Threshold <= 0 {
		config.Threshold = 10
	}
	if config.Window <= 0 {
		config.Window = 60 * time.Second
	}
	if config.CoolDown <= 0 {
		config.CoolDown = 30 * time.Second
	}
	if config.IsFailure == nil {
		config.IsFailure = func(c *gin.Context) bool {
			return c.Writer.Status() >= http.StatusInternalServerError
		}
	}

	cb := newCircuitBreaker(config)

	return func(c *gin.Context) {
		if !cb.allow() {
			if config.OnOpen != nil {
				config.OnOpen(c)
			} else {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"code":    50300,
					"message": "Service Unavailable (Circuit Breaker Open)",
				})
			}
			return
		}

		c.Next()

		if config.IsFailure(c) {
			cb.recordFailure()
		} else {
			cb.recordSuccess()
		}
	}
}
