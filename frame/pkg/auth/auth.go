package auth

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 声明
type Claims struct {
	UserID            uint   `json:"user_id"`
	Username          string `json:"username"`
	Role              string `json:"role,omitempty"`
	ClientID          string `json:"client_id,omitempty"`
	Scope             string `json:"scope,omitempty"`
	IP                string `json:"ip,omitempty"`
	DeviceFingerprint string `json:"device_fingerprint,omitempty"`
	TokenType         string `json:"token_type,omitempty"` // access, refresh, temp
	jwt.RegisteredClaims
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret     string        `json:"secret" yaml:"secret"`
	ExpireTime time.Duration `json:"expire_time" yaml:"expire_time"`
	Issuer     string        `json:"issuer" yaml:"issuer"`
}

// GenerateTokenOption 生成 Token 的可选参数
type GenerateTokenOption struct {
	Role              string
	ClientID          string
	Scope             string
	IP                string
	DeviceFingerprint string
	TokenType         string
	ExpireTime        time.Duration // 为空时使用 JWTConfig 的默认值
}

var (
	globalJWTConfig *JWTConfig
	jwtMu           sync.RWMutex
)

func InitJWT(cfg *JWTConfig) {
	jwtMu.Lock()
	defer jwtMu.Unlock()
	globalJWTConfig = cfg
}

func GetJWTConfig() *JWTConfig {
	jwtMu.RLock()
	defer jwtMu.RUnlock()
	return globalJWTConfig
}

// GenerateToken 生成 JWT Token（简化版）
func GenerateToken(userID uint, username string, role ...string) (string, error) {
	opt := GenerateTokenOption{}
	if len(role) > 0 {
		opt.Role = role[0]
	}
	return GenerateTokenWithOpts(userID, username, opt)
}

// GenerateTokenWithOpts 生成 JWT Token（完整版，支持自定义 claims）
func GenerateTokenWithOpts(userID uint, username string, opt GenerateTokenOption) (string, error) {
	cfg := GetJWTConfig()
	if cfg == nil {
		return "", errors.New("JWT config not initialized")
	}

	expire := opt.ExpireTime
	if expire == 0 {
		expire = cfg.ExpireTime
	}

	claims := Claims{
		UserID:            userID,
		Username:          username,
		Role:              opt.Role,
		ClientID:          opt.ClientID,
		Scope:             opt.Scope,
		IP:                opt.IP,
		DeviceFingerprint: opt.DeviceFingerprint,
		TokenType:         opt.TokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    cfg.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

// ParseToken 解析 JWT Token
func ParseToken(tokenString string) (*Claims, error) {
	cfg := GetJWTConfig()
	if cfg == nil {
		return nil, errors.New("JWT config not initialized")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshToken 刷新 Token（保留原有 optional claims）
func RefreshToken(tokenString string, opts ...GenerateTokenOption) (string, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return "", err
	}

	opt := GenerateTokenOption{
		Role:              claims.Role,
		ClientID:          claims.ClientID,
		Scope:             claims.Scope,
		IP:                claims.IP,
		DeviceFingerprint: claims.DeviceFingerprint,
		TokenType:         claims.TokenType,
	}
	if len(opts) > 0 {
		if opts[0].Role != "" {
			opt.Role = opts[0].Role
		}
		if opts[0].ClientID != "" {
			opt.ClientID = opts[0].ClientID
		}
		// merge other overrides as needed
	}

	return GenerateTokenWithOpts(claims.UserID, claims.Username, opt)
}
