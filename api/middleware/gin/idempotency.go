package gin

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// IdempotencyStore 幂等性存储接口。
// 用户可根据需要选择内存、Redis 等实现。
type IdempotencyStore interface {
	// Get 获取已存储的幂等结果。
	// 如果 key 不存在，返回 (nil, nil)。
	Get(ctx context.Context, key string) (*IdempotencyResult, error)

	// Set 存储幂等结果。
	Set(ctx context.Context, key string, result *IdempotencyResult, ttl time.Duration) error
}

// IdempotencyResult 幂等性请求的存储结果。
type IdempotencyResult struct {
	StatusCode int
	Body       json.RawMessage
	Headers    map[string]string
}

// IdempotencyOption 幂等性中间件配置选项。
type IdempotencyOption func(*idempotencyConfig)

type idempotencyConfig struct {
	headerName string
	ttl        time.Duration
	store      IdempotencyStore
}

// WithIdempotencyHeader 自定义幂等性请求头名称。
func WithIdempotencyHeader(name string) IdempotencyOption {
	return func(c *idempotencyConfig) {
		c.headerName = name
	}
}

// WithIdempotencyTTL 设置幂等 Key 的过期时间。
func WithIdempotencyTTL(ttl time.Duration) IdempotencyOption {
	return func(c *idempotencyConfig) {
		c.ttl = ttl
	}
}

const defaultIdempotencyHeader = "Idempotency-Key"
const defaultIdempotencyTTL = 24 * time.Hour

// Idempotency 幂等性保护中间件。
// 客户端在请求头中传入 Idempotency-Key，服务端缓存处理结果，
// 相同 Key 的重复请求直接返回缓存结果。
//
// 用法:
//
//	store := ginmw.NewMemoryIdempotencyStore()
//	r.Use(ginmw.Idempotency(store))
func Idempotency(store IdempotencyStore, opts ...IdempotencyOption) gin.HandlerFunc {
	cfg := &idempotencyConfig{
		headerName: defaultIdempotencyHeader,
		ttl:        defaultIdempotencyTTL,
		store:      store,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(c *gin.Context) {
		// GET 请求无需幂等性保护
		if c.Request.Method == http.MethodGet {
			c.Next()
			return
		}

		key := c.GetHeader(cfg.headerName)
		if key == "" {
			c.Next()
			return
		}

		// 查询是否已处理
		if result, err := store.Get(c.Request.Context(), key); err == nil && result != nil {
			for k, v := range result.Headers {
				c.Header(k, v)
			}
			c.AbortWithStatusJSON(result.StatusCode, result.Body)
			return
		}

		// 记录请求标识，供后续 handler 保存结果
		c.Set("_idempotency_key", key)
		c.Set("_idempotency_store", store)
		c.Set("_idempotency_ttl", cfg.ttl)
		c.Next()
	}
}

// SaveIdempotencyResult 保存幂等性结果（由业务 handler 在返回时调用）。
// 框架的自动响应处理可集成此函数。用户也可自己在 handler 中手动调用。
func SaveIdempotencyResult(c *gin.Context, statusCode int, body interface{}) {
	key, exists := c.Get("_idempotency_key")
	if !exists {
		return
	}

	store, exists := c.Get("_idempotency_store")
	if !exists {
		return
	}

	ttl, exists := c.Get("_idempotency_ttl")
	if !exists {
		return
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return
	}

	_ = store.(IdempotencyStore).Set(
		c.Request.Context(),
		key.(string),
		&IdempotencyResult{
			StatusCode: statusCode,
			Body:       bodyBytes,
		},
		ttl.(time.Duration),
	)
}

// ==================== 内存实现 ====================

type memoryIdempotencyStore struct {
	mu   sync.RWMutex
	data map[string]*idempotencyEntry
}

type idempotencyEntry struct {
	result    *IdempotencyResult
	expiresAt time.Time
}

// NewMemoryIdempotencyStore 创建基于内存的幂等性存储。
// 适用于单实例部署，多实例场景请使用 Redis 实现。
func NewMemoryIdempotencyStore() IdempotencyStore {
	s := &memoryIdempotencyStore{
		data: make(map[string]*idempotencyEntry),
	}
	// 定期清理过期 key
	go s.cleanup()
	return s
}

func (s *memoryIdempotencyStore) Get(_ context.Context, key string) (*IdempotencyResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.data[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, nil
	}
	return entry.result, nil
}

func (s *memoryIdempotencyStore) Set(_ context.Context, key string, result *IdempotencyResult, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = &idempotencyEntry{
		result:    result,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (s *memoryIdempotencyStore) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for k, v := range s.data {
			if now.After(v.expiresAt) {
				delete(s.data, k)
			}
		}
		s.mu.Unlock()
	}
}
