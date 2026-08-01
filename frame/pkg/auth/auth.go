package auth

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
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
	// Secret HMAC（HS256）对称密钥，仅 SigningMethod 为 HS256 时使用
	Secret     string        `json:"secret" yaml:"secret"`
	ExpireTime time.Duration `json:"expire_time" yaml:"expire_time"`
	Issuer     string        `json:"issuer" yaml:"issuer"`

	// SigningMethod 签名算法：HS256（默认）/ RS256 / ES256
	// 留空时使用 HS256，保证向后兼容
	SigningMethod string `json:"signing_method" yaml:"signing_method"`

	// PrivateKeyPath 非对称签名私钥 PEM 文件路径（RS256/ES256）
	// 优先级高于 PrivateKey 字段
	PrivateKeyPath string `json:"private_key_path" yaml:"private_key_path"`
	// PublicKeyPath 非对称验签公钥 PEM 文件路径（RS256/ES256）
	// 未配置时自动从私钥推导公钥；验签专用场景（如仅做校验的服务）应显式配置
	PublicKeyPath string `json:"public_key_path" yaml:"public_key_path"`

	// PrivateKey 直接注入的非对称私钥（*rsa.PrivateKey / *ecdsa.PrivateKey）
	// 供程序化配置使用，不进 JSON/YAML 序列化
	PrivateKey any `json:"-" yaml:"-"`
	// PublicKey 直接注入的非对称公钥（*rsa.PublicKey / *ecdsa.PublicKey）
	PublicKey any `json:"-" yaml:"-"`
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
	// 配置更新后清空密钥缓存，避免残留旧密钥
	resetKeyCache()
	globalJWTConfig = cfg
}

func GetJWTConfig() *JWTConfig {
	jwtMu.RLock()
	defer jwtMu.RUnlock()
	return globalJWTConfig
}

// minSecretLength JWT 密钥最小长度（防止弱密钥被暴力破解）
const minSecretLength = 32

// ValidateSecret 校验 HS256 对称密钥强度
// 密钥必须非空且长度不少于 32 字符，否则拒绝签发/解析 Token
func ValidateSecret(secret string) error {
	if secret == "" {
		return errors.New("JWT secret is not configured: set LANDC_JWT_SECRET environment variable or jwt.secret in config")
	}
	if len(secret) < minSecretLength {
		return fmt.Errorf("JWT secret is too weak: expected at least %d characters, got %d", minSecretLength, len(secret))
	}
	return nil
}

// validateConfig 按签名算法校验配置：
// - HS256：校验对称密钥强度
// - RS256/ES256：校验非对称私钥可加载
func validateConfig(cfg *JWTConfig) error {
	method, err := cfg.signingMethod()
	if err != nil {
		return err
	}

	switch method.(type) {
	case *jwt.SigningMethodHMAC:
		return ValidateSecret(cfg.Secret)
	case *jwt.SigningMethodRSA, *jwt.SigningMethodECDSA:
		_, err := cfg.signingKey(method)
		return err
	default:
		return fmt.Errorf("unsupported signing method: %s", cfg.SigningMethod)
	}
}

// signingMethod 解析配置的签名算法，默认 HS256
func (c *JWTConfig) signingMethod() (jwt.SigningMethod, error) {
	method := strings.ToUpper(strings.TrimSpace(c.SigningMethod))
	if method == "" {
		method = "HS256"
	}
	switch method {
	case "HS256":
		return jwt.SigningMethodHS256, nil
	case "RS256":
		return jwt.SigningMethodRS256, nil
	case "ES256":
		return jwt.SigningMethodES256, nil
	default:
		return nil, fmt.Errorf("unsupported signing method: %s", c.SigningMethod)
	}
}

// signingKey 返回签名密钥（HS256: []byte；RS256: *rsa.PrivateKey；ES256: *ecdsa.PrivateKey）
func (c *JWTConfig) signingKey(method jwt.SigningMethod) (any, error) {
	switch method.(type) {
	case *jwt.SigningMethodHMAC:
		return []byte(c.Secret), nil
	case *jwt.SigningMethodRSA, *jwt.SigningMethodECDSA:
		if c.PrivateKeyPath != "" {
			return loadPEMKey(c.PrivateKeyPath, true)
		}
		if c.PrivateKey != nil {
			return c.PrivateKey, nil
		}
		return nil, errors.New("private key not configured: set jwt.private_key_path or provide PrivateKey")
	default:
		return nil, fmt.Errorf("unsupported signing method: %s", c.SigningMethod)
	}
}

// verifyingKey 返回验签密钥（HS256: []byte；RS256: *rsa.PublicKey；ES256: *ecdsa.PublicKey）
func (c *JWTConfig) verifyingKey(method jwt.SigningMethod) (any, error) {
	switch method.(type) {
	case *jwt.SigningMethodHMAC:
		return []byte(c.Secret), nil
	case *jwt.SigningMethodRSA, *jwt.SigningMethodECDSA:
		if c.PublicKeyPath != "" {
			return loadPEMKey(c.PublicKeyPath, false)
		}
		if c.PublicKey != nil {
			return c.PublicKey, nil
		}
		// 未显式配置公钥时，从私钥推导（签发方场景）
		return c.publicKeyFromPrivate()
	default:
		return nil, fmt.Errorf("unsupported signing method: %s", c.SigningMethod)
	}
}

// publicKeyFromPrivate 从私钥推导公钥
func (c *JWTConfig) publicKeyFromPrivate() (any, error) {
	priv, err := c.signingKey(&jwt.SigningMethodRSA{})
	if err != nil {
		return nil, errors.New("public key not configured: set jwt.public_key_path or provide PublicKey")
	}
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey, nil
	case *ecdsa.PrivateKey:
		return &k.PublicKey, nil
	}
	return nil, errors.New("unsupported private key type for public key derivation")
}

// GenerateToken 生成 JWT Token（简化版）
func GenerateToken(userID uint, username string, role ...string) (string, error) {
	opt := GenerateTokenOption{}
	if len(role) > 0 {
		opt.Role = role[0]
	}
	return GenerateTokenWithOpts(userID, username, opt)
}

// GenerateTokenWithOpts 生成 JWT Token（完整版，支持自定义 claims 与签名算法）
func GenerateTokenWithOpts(userID uint, username string, opt GenerateTokenOption) (string, error) {
	cfg := GetJWTConfig()
	if cfg == nil {
		return "", errors.New("JWT config not initialized")
	}

	if err := validateConfig(cfg); err != nil {
		return "", err
	}

	method, err := cfg.signingMethod()
	if err != nil {
		return "", err
	}
	signingKey, err := cfg.signingKey(method)
	if err != nil {
		return "", err
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

	token := jwt.NewWithClaims(method, claims)
	return token.SignedString(signingKey)
}

// ParseToken 解析并验证 JWT Token
// 通过算法白名单（WithValidMethods）防止算法混淆攻击
func ParseToken(tokenString string) (*Claims, error) {
	cfg := GetJWTConfig()
	if cfg == nil {
		return nil, errors.New("JWT config not initialized")
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	method, err := cfg.signingMethod()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 双重校验：比对 alg 字符串 + 依赖 WithValidMethods 白名单
		if token.Method.Alg() != method.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return cfg.verifyingKey(method)
	}, jwt.WithValidMethods([]string{method.Alg()}))

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

// ==================== PEM 密钥加载（带缓存） ====================

type keyCacheEntry struct {
	key any
	err error
}

var (
	keyCache = make(map[string]keyCacheEntry)
	keyMu    sync.Mutex
)

// resetKeyCache 清空密钥缓存（InitJWT 配置变更时调用）
func resetKeyCache() {
	keyMu.Lock()
	defer keyMu.Unlock()
	keyCache = make(map[string]keyCacheEntry)
}

// loadPEMKey 加载 PEM 密钥文件（按路径缓存）
func loadPEMKey(path string, isPrivate bool) (any, error) {
	keyMu.Lock()
	defer keyMu.Unlock()
	if entry, ok := keyCache[path]; ok {
		return entry.key, entry.err
	}
	key, err := parsePEMFile(path, isPrivate)
	keyCache[path] = keyCacheEntry{key: key, err: err}
	return key, err
}

// parsePEMFile 解析 PEM 文件中的密钥
// 私钥支持：PKCS#1 RSA / SEC1 EC / PKCS#8（RSA 与 EC）
// 公钥支持：PKIX / PKCS#1 RSA / X.509 证书
func parsePEMFile(path string, isPrivate bool) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file %s: %w", path, err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM data in %s", path)
	}

	if isPrivate {
		switch block.Type {
		case "RSA PRIVATE KEY":
			return x509.ParsePKCS1PrivateKey(block.Bytes)
		case "EC PRIVATE KEY":
			return x509.ParseECPrivateKey(block.Bytes)
		case "PRIVATE KEY": // PKCS#8
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, err
			}
			switch key.(type) {
			case *rsa.PrivateKey, *ecdsa.PrivateKey:
				return key, nil
			default:
				return nil, fmt.Errorf("unsupported PKCS#8 private key type: %T", key)
			}
		default:
			return nil, fmt.Errorf("unsupported private key PEM type: %s", block.Type)
		}
	}

	// 公钥 / 证书
	switch block.Type {
	case "PUBLIC KEY": // PKIX
		return x509.ParsePKIXPublicKey(block.Bytes)
	case "RSA PUBLIC KEY": // PKCS#1
		return x509.ParsePKCS1PublicKey(block.Bytes)
	case "CERTIFICATE":
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		switch cert.PublicKey.(type) {
		case *rsa.PublicKey, *ecdsa.PublicKey:
			return cert.PublicKey, nil
		default:
			return nil, fmt.Errorf("unsupported certificate public key type: %T", cert.PublicKey)
		}
	default:
		return nil, fmt.Errorf("unsupported public key PEM type: %s", block.Type)
	}
}
