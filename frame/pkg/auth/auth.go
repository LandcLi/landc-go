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
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret     string        `json:"secret" yaml:"secret"`
	ExpireTime time.Duration `json:"expire_time" yaml:"expire_time"`
	Issuer     string        `json:"issuer" yaml:"issuer"`
}

var (
	globalJWTConfig *JWTConfig
	jwtMu           sync.RWMutex
)

// InitJWT 初始化 JWT 配置
func InitJWT(cfg *JWTConfig) {
	jwtMu.Lock()
	defer jwtMu.Unlock()
	globalJWTConfig = cfg
}

// GetJWTConfig 获取 JWT 配置
func GetJWTConfig() *JWTConfig {
	jwtMu.RLock()
	defer jwtMu.RUnlock()
	return globalJWTConfig
}

// GenerateToken 生成 JWT Token
func GenerateToken(userID uint, username string, role ...string) (string, error) {
	cfg := GetJWTConfig()
	if cfg == nil {
		return "", errors.New("JWT config not initialized")
	}

	userRole := ""
	if len(role) > 0 {
		userRole = role[0]
	}

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     userRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.ExpireTime)),
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

// RefreshToken 刷新 Token
func RefreshToken(tokenString string) (string, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return "", err
	}
	return GenerateToken(claims.UserID, claims.Username, claims.Role)
}
