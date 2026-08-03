package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func genRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	return key
}

func genECDSAKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ecdsa key: %v", err)
	}
	return key
}

// TestGenerateParseTokenRS256 验证 RS256 签发与验签（公钥显式注入）
func TestGenerateParseTokenRS256(t *testing.T) {
	priv := genRSAKey(t)

	InitJWT(&JWTConfig{
		SigningMethod: "RS256",
		PrivateKey:    priv,
		PublicKey:     &priv.PublicKey,
		ExpireTime:    1 * time.Hour,
		Issuer:        "test",
	})

	token, err := GenerateToken(1, "rsa-user", "admin")
	if err != nil {
		t.Fatalf("GenerateToken RS256 failed: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken RS256 failed: %v", err)
	}
	if claims.UserID != 1 || claims.Username != "rsa-user" {
		t.Errorf("unexpected claims: %+v", claims)
	}
}

// TestGenerateParseTokenRS256PublicKeyFromPrivate 验证未配置公钥时从私钥推导
func TestGenerateParseTokenRS256PublicKeyFromPrivate(t *testing.T) {
	priv := genRSAKey(t)

	InitJWT(&JWTConfig{
		SigningMethod: "RS256",
		PrivateKey:    priv,
		ExpireTime:    1 * time.Hour,
		Issuer:        "test",
	})

	token, err := GenerateToken(1, "user", "")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if _, err := ParseToken(token); err != nil {
		t.Fatalf("ParseToken with derived public key failed: %v", err)
	}
}

// TestGenerateParseTokenES256 验证 ES256 签发与验签
func TestGenerateParseTokenES256(t *testing.T) {
	priv := genECDSAKey(t)

	InitJWT(&JWTConfig{
		SigningMethod: "ES256",
		PrivateKey:    priv,
		PublicKey:     &priv.PublicKey,
		ExpireTime:    1 * time.Hour,
		Issuer:        "test",
	})

	token, err := GenerateToken(2, "ecdsa-user", "user")
	if err != nil {
		t.Fatalf("GenerateToken ES256 failed: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken ES256 failed: %v", err)
	}
	if claims.UserID != 2 {
		t.Errorf("unexpected user id: %d", claims.UserID)
	}
}

// TestRS256AlgorithmConfusionAttack 验证算法混淆攻击被拒绝：
// RS256 配置下，攻击者用 HS256 + 公钥内容伪造的 token 必须验签失败
func TestRS256AlgorithmConfusionAttack(t *testing.T) {
	priv := genRSAKey(t)

	InitJWT(&JWTConfig{
		SigningMethod: "RS256",
		PrivateKey:    priv,
		PublicKey:     &priv.PublicKey,
		ExpireTime:    1 * time.Hour,
		Issuer:        "test",
	})

	// 攻击者用 HS256 以公钥 DER 内容为对称密钥伪造 token
	pubDER := x509.MarshalPKCS1PublicKey(&priv.PublicKey)
	attackerClaims := Claims{
		UserID:   999,
		Username: "attacker",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    "test",
		},
	}
	forged, err := jwt.NewWithClaims(jwt.SigningMethodHS256, attackerClaims).
		SignedString(pubDER)
	if err != nil {
		t.Fatalf("forge token: %v", err)
	}

	// RS256 模式下解析该 HS256 伪造 token 必须失败
	if _, err := ParseToken(forged); err == nil {
		t.Fatal("algorithm confusion attack should be rejected")
	}
}

// TestWrongAlgorithmRejected 验证用不匹配算法签发的 token 被拒绝
func TestWrongAlgorithmRejected(t *testing.T) {
	// 用 HS256 签发，RS256 验签
	hsConfig := JWTConfig{
		Secret:     "test-secret-key-0123456789abcdef0123456789abcdef",
		ExpireTime: 1 * time.Hour,
		Issuer:     "test",
	}
	InitJWT(&hsConfig)
	hsToken, err := GenerateToken(1, "user")
	if err != nil {
		t.Fatalf("generate HS256 token: %v", err)
	}

	// 切到 RS256 配置
	priv := genRSAKey(t)
	InitJWT(&JWTConfig{
		SigningMethod: "RS256",
		PrivateKey:    priv,
		PublicKey:     &priv.PublicKey,
		ExpireTime:    1 * time.Hour,
		Issuer:        "test",
	})
	if _, err := ParseToken(hsToken); err == nil {
		t.Fatal("HS256 token must not verify under RS256 config")
	}
}

// TestPEMFileLoading 验证从 PEM 文件加载 RSA 密钥
func TestPEMFileLoading(t *testing.T) {
	priv := genRSAKey(t)

	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: mustMarshalPKIX(&priv.PublicKey)})

	dir := t.TempDir()
	privPath := filepath.Join(dir, "private.pem")
	pubPath := filepath.Join(dir, "public.pem")
	if err := os.WriteFile(privPath, privPEM, 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	if err := os.WriteFile(pubPath, pubPEM, 0o600); err != nil {
		t.Fatalf("write public key: %v", err)
	}

	InitJWT(&JWTConfig{
		SigningMethod:  "RS256",
		PrivateKeyPath: privPath,
		PublicKeyPath:  pubPath,
		ExpireTime:     1 * time.Hour,
		Issuer:         "test",
	})

	token, err := GenerateToken(5, "file-user")
	if err != nil {
		t.Fatalf("GenerateToken with PEM files: %v", err)
	}
	if _, err := ParseToken(token); err != nil {
		t.Fatalf("ParseToken with PEM files: %v", err)
	}
}

// TestPEMInvalidPath 验证私钥路径不存在时报错
func TestPEMInvalidPath(t *testing.T) {
	InitJWT(&JWTConfig{
		SigningMethod:  "RS256",
		PrivateKeyPath: "/nonexistent/private.pem",
		ExpireTime:     1 * time.Hour,
	})

	if _, err := GenerateToken(1, "user"); err == nil {
		t.Fatal("GenerateToken should fail when private key file missing")
	}
}

// TestPrivateKeyInsecurePermissions 验证私钥文件权限过宽时被拒绝
func TestPrivateKeyInsecurePermissions(t *testing.T) {
	priv := genRSAKey(t)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	dir := t.TempDir()
	// 0644：同组/其他用户可读，应被拒绝
	insecurePath := filepath.Join(dir, "insecure.pem")
	if err := os.WriteFile(insecurePath, privPEM, 0o644); err != nil {
		t.Fatalf("write private key: %v", err)
	}

	InitJWT(&JWTConfig{
		SigningMethod:  "RS256",
		PrivateKeyPath: insecurePath,
		ExpireTime:     1 * time.Hour,
		Issuer:         "test",
	})

	if _, err := GenerateToken(1, "user"); err == nil {
		t.Fatal("GenerateToken should fail when private key permissions are too loose")
	}

	// 0600 的另一路径：权限合法应成功
	securePath := filepath.Join(dir, "secure.pem")
	if err := os.WriteFile(securePath, privPEM, 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	InitJWT(&JWTConfig{
		SigningMethod:  "RS256",
		PrivateKeyPath: securePath,
		ExpireTime:     1 * time.Hour,
		Issuer:         "test",
	})
	token, err := GenerateToken(1, "user")
	if err != nil {
		t.Fatalf("GenerateToken with 0600 key should succeed: %v", err)
	}
	if _, err := ParseToken(token); err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
}

// TestUnsupportedSigningMethod 验证不支持的算法被拒绝
func TestUnsupportedSigningMethod(t *testing.T) {
	InitJWT(&JWTConfig{
		SigningMethod: "HS512",
		Secret:        "test-secret-key-0123456789abcdef0123456789abcdef",
		ExpireTime:    1 * time.Hour,
	})

	if _, err := GenerateToken(1, "user"); err == nil {
		t.Fatal("GenerateToken should fail for unsupported signing method")
	}
}

// TestRS256WithoutPrivateKey 验证 RS256 缺少私钥时报错
func TestRS256WithoutPrivateKey(t *testing.T) {
	InitJWT(&JWTConfig{
		SigningMethod: "RS256",
		Secret:        "test-secret-key-0123456789abcdef0123456789abcdef",
		ExpireTime:    1 * time.Hour,
	})

	if _, err := GenerateToken(1, "user"); err == nil {
		t.Fatal("GenerateToken should fail without RSA private key")
	}
}

// TestRefreshTokenRS256 验证 RS256 模式下刷新 token
func TestRefreshTokenRS256(t *testing.T) {
	priv := genRSAKey(t)
	InitJWT(&JWTConfig{
		SigningMethod: "RS256",
		PrivateKey:    priv,
		PublicKey:     &priv.PublicKey,
		ExpireTime:    1 * time.Hour,
		Issuer:        "test",
	})

	token, err := GenerateToken(1, "user", "admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	newToken, err := RefreshToken(token)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	claims, err := ParseToken(newToken)
	if err != nil {
		t.Fatalf("ParseToken refreshed: %v", err)
	}
	if claims.Role != "admin" {
		t.Errorf("expected role admin, got %q", claims.Role)
	}
}

func mustMarshalPKIX(pub *rsa.PublicKey) []byte {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		panic(err)
	}
	return der
}
