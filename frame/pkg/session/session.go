package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/LandcLi/landc-go/tools/generate"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Store Session 存储接口
type Store interface {
	Get(ctx context.Context, sessionID string) (map[string]interface{}, error)
	Set(ctx context.Context, sessionID string, data map[string]interface{}, maxAge time.Duration) error
	Delete(ctx context.Context, sessionID string) error
	Exists(ctx context.Context, sessionID string) (bool, error)
}

// Config Session 配置
type Config struct {
	// CookieName cookie 名称（默认 "landc_session"）
	CookieName string
	// MaxAge 过期时间（默认 24 小时）
	MaxAge time.Duration
	// Path cookie 路径（默认 "/"）
	Path string
	// Domain cookie 域名
	Domain string
	// Secure 仅 HTTPS
	Secure bool
	// HttpOnly 禁止 JS 访问
	HttpOnly bool
	// SameSite cookie SameSite 策略
	SameSite http.SameSite
}

func defaultConfig() *Config {
	return &Config{
		CookieName: "landc_session",
		MaxAge:     24 * time.Hour,
		Path:       "/",
		HttpOnly:   true,
		SameSite:   http.SameSiteLaxMode,
	}
}

// Session 单个会话
type Session struct {
	ID       string
	data     map[string]interface{}
	modified bool
	mu       sync.RWMutex
}

// Get 获取值
func (s *Session) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

// GetString 获取字符串值
func (s *Session) GetString(key string) string {
	val, ok := s.Get(key)
	if !ok {
		return ""
	}
	str, _ := val.(string)
	return str
}

// GetInt 获取整数值
func (s *Session) GetInt(key string) int {
	val, ok := s.Get(key)
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	}
	return 0
}

// Set 设置值
func (s *Session) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	s.modified = true
}

// Delete 删除值
func (s *Session) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	s.modified = true
}

// Clear 清空所有数据
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]interface{})
	s.modified = true
}

// All 获取所有数据副本
func (s *Session) All() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]interface{}, len(s.data))
	for k, v := range s.data {
		cp[k] = v
	}
	return cp
}

// --- Gin 中间件 ---

const contextKey = "landc_session"

// Middleware 创建 Session 中间件
func Middleware(store Store, config *Config) gin.HandlerFunc {
	if config == nil {
		config = defaultConfig()
	}
	if config.CookieName == "" {
		config.CookieName = "landc_session"
	}
	if config.MaxAge <= 0 {
		config.MaxAge = 24 * time.Hour
	}
	if config.Path == "" {
		config.Path = "/"
	}

	return func(c *gin.Context) {
		// 读取或创建 session
		sessionID := getSessionID(c, config)
		isNew := false
		if sessionID == "" {
			sessionID = generate.UUID()
			isNew = true
		}

		// 加载数据
		var data map[string]interface{}
		if !isNew {
			var err error
			data, err = store.Get(c.Request.Context(), sessionID)
			if err != nil || data == nil {
				data = make(map[string]interface{})
			}
		} else {
			data = make(map[string]interface{})
		}

		sess := &Session{
			ID:   sessionID,
			data: data,
		}

		c.Set(contextKey, sess)
		c.Next()

		// 请求结束后保存
		if sess.modified || isNew {
			sess.mu.RLock()
			saveData := make(map[string]interface{}, len(sess.data))
			for k, v := range sess.data {
				saveData[k] = v
			}
			sess.mu.RUnlock()

			store.Set(c.Request.Context(), sessionID, saveData, config.MaxAge)

			// 设置 cookie
			maxAgeSeconds := int(config.MaxAge.Seconds())
			c.SetSameSite(config.SameSite)
			c.SetCookie(config.CookieName, sessionID, maxAgeSeconds, config.Path, config.Domain, config.Secure, config.HttpOnly)
		}
	}
}

// FromContext 从 Gin Context 获取 Session
func FromContext(c *gin.Context) *Session {
	val, exists := c.Get(contextKey)
	if !exists {
		return nil
	}
	sess, _ := val.(*Session)
	return sess
}

// Destroy 销毁 Session
func Destroy(c *gin.Context, store Store, config *Config) error {
	if config == nil {
		config = defaultConfig()
	}

	sess := FromContext(c)
	if sess == nil {
		return nil
	}

	if err := store.Delete(c.Request.Context(), sess.ID); err != nil {
		return err
	}

	// 删除 cookie
	c.SetCookie(config.CookieName, "", -1, config.Path, config.Domain, config.Secure, config.HttpOnly)
	return nil
}

func getSessionID(c *gin.Context, config *Config) string {
	cookie, err := c.Cookie(config.CookieName)
	if err != nil {
		return ""
	}
	return cookie
}

// --- Redis Store ---

// RedisStore Redis 实现的 Session 存储
type RedisStore struct {
	client *redis.Client
	prefix string
}

// NewRedisStore 创建 Redis Session 存储
func NewRedisStore(client *redis.Client, prefix string) *RedisStore {
	if prefix == "" {
		prefix = "session:"
	}
	return &RedisStore{client: client, prefix: prefix}
}

func (s *RedisStore) key(sessionID string) string {
	return s.prefix + sessionID
}

func (s *RedisStore) Get(ctx context.Context, sessionID string) (map[string]interface{}, error) {
	data, err := s.client.Get(ctx, s.key(sessionID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *RedisStore) Set(ctx context.Context, sessionID string, data map[string]interface{}, maxAge time.Duration) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.key(sessionID), jsonData, maxAge).Err()
}

func (s *RedisStore) Delete(ctx context.Context, sessionID string) error {
	return s.client.Del(ctx, s.key(sessionID)).Err()
}

func (s *RedisStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	n, err := s.client.Exists(ctx, s.key(sessionID)).Result()
	return n > 0, err
}

// --- Memory Store (适用于开发/测试) ---

// MemoryStore 内存实现的 Session 存储
type MemoryStore struct {
	data map[string]*memoryEntry
	mu   sync.RWMutex
}

type memoryEntry struct {
	data      map[string]interface{}
	expiresAt time.Time
}

// NewMemoryStore 创建内存 Session 存储
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		data: make(map[string]*memoryEntry),
	}
	go store.cleanup()
	return store
}

func (s *MemoryStore) Get(_ context.Context, sessionID string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.data[sessionID]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, nil
	}

	// 返回副本
	cp := make(map[string]interface{}, len(entry.data))
	for k, v := range entry.data {
		cp[k] = v
	}
	return cp, nil
}

func (s *MemoryStore) Set(_ context.Context, sessionID string, data map[string]interface{}, maxAge time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[sessionID] = &memoryEntry{
		data:      data,
		expiresAt: time.Now().Add(maxAge),
	}
	return nil
}

func (s *MemoryStore) Delete(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, sessionID)
	return nil
}

func (s *MemoryStore) Exists(_ context.Context, sessionID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.data[sessionID]
	if !ok || time.Now().After(entry.expiresAt) {
		return false, nil
	}
	return true, nil
}

func (s *MemoryStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, entry := range s.data {
			if now.After(entry.expiresAt) {
				delete(s.data, id)
			}
		}
		s.mu.Unlock()
	}
}

// Size 获取活跃 session 数（用于监控）
func (s *MemoryStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	now := time.Now()
	for _, entry := range s.data {
		if now.Before(entry.expiresAt) {
			count++
		}
	}
	return count
}

// --- 辅助函数 ---

// MustFromContext 从 Context 获取 Session，不存在则 panic
func MustFromContext(c *gin.Context) *Session {
	sess := FromContext(c)
	if sess == nil {
		panic(fmt.Sprintf("session not found in context, ensure session.Middleware is registered"))
	}
	return sess
}
