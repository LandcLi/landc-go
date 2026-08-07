// Package verifycode 提供一次性验证码的生成、存储与校验。
//
// 依赖注入：存储与限流共用调用方注入的 tools/ratelimit.Cache 实现
// （Redis 或本地），本包不依赖任何框架。
// 语义：
//   - 生成：受发送间隔（默认 60s）与每日上限（默认 10 次）约束
//   - 存储：TTL 默认 5 分钟
//   - 校验：一次性（成功后立即删除），防重放
package verifycode

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/LandcLi/landc-go/tools/ratelimit"
)

// 错误定义。
var (
	// ErrIntervalLimited 发送间隔未到。
	ErrIntervalLimited = errors.New("verifycode: send interval limited")
	// ErrDailyLimited 超出每日发送上限。
	ErrDailyLimited = errors.New("verifycode: daily limit exceeded")
)

// Manager 验证码管理器。
type Manager struct {
	cache        ratelimit.Cache
	interval     *ratelimit.IntervalLimiter
	count        *ratelimit.CountLimiter
	codeLength   int
	ttl          time.Duration
	sendInterval time.Duration
	dailyLimit   int64
}

// Option 配置项。
type Option func(*Manager)

// WithCodeLength 设置验证码长度（默认 6）。
func WithCodeLength(n int) Option {
	return func(m *Manager) {
		if n > 0 {
			m.codeLength = n
		}
	}
}

// WithTTL 设置验证码有效期（默认 5 分钟）。
func WithTTL(d time.Duration) Option {
	return func(m *Manager) {
		if d > 0 {
			m.ttl = d
		}
	}
}

// WithSendInterval 设置同一 key 的发送间隔（默认 60 秒）。
func WithSendInterval(d time.Duration) Option {
	return func(m *Manager) {
		if d > 0 {
			m.sendInterval = d
		}
	}
}

// WithDailyLimit 设置同一 key 的每日发送上限（默认 10）。
func WithDailyLimit(n int64) Option {
	return func(m *Manager) {
		if n > 0 {
			m.dailyLimit = n
		}
	}
}

// NewManager 创建验证码管理器。
func NewManager(c ratelimit.Cache, opts ...Option) *Manager {
	m := &Manager{
		cache:        c,
		interval:     ratelimit.NewIntervalLimiter(c),
		count:        ratelimit.NewCountLimiter(c),
		codeLength:   6,
		ttl:          5 * time.Minute,
		sendInterval: 60 * time.Second,
		dailyLimit:   10,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Generate 生成验证码并存储，返回验证码值（供业务侧发送短信/邮件）。
// 受发送间隔与每日上限约束；超限返回 ErrIntervalLimited / ErrDailyLimited。
func (m *Manager) Generate(key string) (string, error) {
	if !m.interval.Allow("vc:interval:"+key, m.sendInterval) {
		return "", ErrIntervalLimited
	}
	if !m.count.Allow("vc:daily:"+key, m.dailyLimit, 24*time.Hour) {
		return "", ErrDailyLimited
	}

	code, err := randomCode(m.codeLength)
	if err != nil {
		return "", err
	}
	if err := m.cache.Set("vc:code:"+key, code, m.ttl); err != nil {
		return "", fmt.Errorf("verifycode: store code: %w", err)
	}
	return code, nil
}

// Verify 校验验证码：成功则删除（一次性，防重放），返回 true。
func (m *Manager) Verify(key, input string) bool {
	stored, err := m.cache.Get("vc:code:" + key)
	if err != nil || stored == "" {
		return false
	}
	if stored != input {
		return false
	}
	_ = m.cache.Delete("vc:code:" + key)
	return true
}

// randomCode 使用 crypto/rand 生成 length 位数字验证码。
func randomCode(length int) (string, error) {
	buf := make([]byte, length)
	maxVal := big.NewInt(10)
	for i := range buf {
		n, err := rand.Int(rand.Reader, maxVal)
		if err != nil {
			return "", fmt.Errorf("verifycode: generate code: %w", err)
		}
		//nolint:gosec // n 由 big.Int(10) 限定为 0-9，int64->byte 无溢出风险
		buf[i] = byte('0' + n.Int64())
	}
	return string(buf), nil
}
