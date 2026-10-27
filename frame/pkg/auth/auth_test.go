package auth

import (
	"testing"
	"time"
)

func TestGenerateAndParseToken(t *testing.T) {
	InitJWT(&JWTConfig{
		Secret:     "test-secret-key-0123456789abcdef0123456789abcdef",
		ExpireTime: 1 * time.Hour,
		Issuer:     "test",
	})

	token, err := GenerateToken(1, "testuser", "admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	if claims.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", claims.UserID)
	}
	if claims.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got '%s'", claims.Username)
	}
	if claims.Role != "admin" {
		t.Errorf("Expected Role 'admin', got '%s'", claims.Role)
	}
}

func TestParseInvalidToken(t *testing.T) {
	InitJWT(&JWTConfig{
		Secret:     "test-secret-key-0123456789abcdef0123456789abcdef",
		ExpireTime: 1 * time.Hour,
		Issuer:     "test",
	})

	_, err := ParseToken("invalid-token")
	if err == nil {
		t.Error("ParseToken should fail for invalid token")
	}
}

func TestExpiredToken(t *testing.T) {
	InitJWT(&JWTConfig{
		Secret:     "test-secret-key-0123456789abcdef0123456789abcdef",
		ExpireTime: -1 * time.Hour, // 已过期
		Issuer:     "test",
	})

	token, err := GenerateToken(1, "testuser")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = ParseToken(token)
	if err == nil {
		t.Error("ParseToken should fail for expired token")
	}
}

func TestRefreshToken(t *testing.T) {
	InitJWT(&JWTConfig{
		Secret:     "test-secret-key-0123456789abcdef0123456789abcdef",
		ExpireTime: 1 * time.Hour,
		Issuer:     "test",
	})

	token, _ := GenerateToken(1, "testuser", "user")
	newToken, err := RefreshToken(token)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	if newToken == "" {
		t.Fatal("new token should not be empty")
	}

	claims, err := ParseToken(newToken)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", claims.UserID)
	}
}

func TestGenerateTokenWithoutConfig(t *testing.T) {
	// 重置配置
	jwtMu.Lock()
	globalJWTConfig = nil
	jwtMu.Unlock()

	_, err := GenerateToken(1, "testuser")
	if err == nil {
		t.Error("GenerateToken should fail without config")
	}
}

func TestGenerateTokenWithoutRole(t *testing.T) {
	InitJWT(&JWTConfig{
		Secret:     "test-secret-key-0123456789abcdef0123456789abcdef",
		ExpireTime: 1 * time.Hour,
		Issuer:     "test",
	})

	token, err := GenerateToken(1, "testuser")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.Role != "" {
		t.Errorf("Expected empty role, got '%s'", claims.Role)
	}
}

// TestGenerateTokenUniquePerCall 令牌必须每次唯一。
//
// 缺陷背景：签发时没有填写 jti（RegisteredClaims.ID），于是同一用户在同一秒内
// 多次签发得到的 claims 完全相同（user_id / scope / ip / 指纹 / iat 一致），
// 签名也就一致 —— 两次登录拿到一模一样的字符串。
//
// 实测后果（LandcSSO 的并发会话限制）：
//   - 按 access_token 查库永远命中最早那条记录，撤销一个等于撤销全部；
//   - 「踢掉最旧的会话」无法区分会话，形同虚设。
func TestGenerateTokenUniquePerCall(t *testing.T) {
	InitJWT(&JWTConfig{
		Secret:     "test-secret-key-0123456789abcdef0123456789abcdef",
		ExpireTime: 1 * time.Hour,
		Issuer:     "test",
	})

	// 连续签发两次（通常落在同一秒内，这正是缺陷的触发条件）
	first, err := GenerateTokenWithOpts(7, "uniq", GenerateTokenOption{TokenType: "access"})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := GenerateTokenWithOpts(7, "uniq", GenerateTokenOption{TokenType: "access"})
	if err != nil {
		t.Fatalf("second: %v", err)
	}

	if first == second {
		t.Fatal("两次签发得到完全相同的令牌（jti 未生效）")
	}

	c1, err := ParseToken(first)
	if err != nil {
		t.Fatalf("parse first: %v", err)
	}
	c2, err := ParseToken(second)
	if err != nil {
		t.Fatalf("parse second: %v", err)
	}
	if c1.ID == "" || c2.ID == "" {
		t.Fatal("jti 为空，令牌无法唯一标识")
	}
	if c1.ID == c2.ID {
		t.Fatalf("两个令牌的 jti 相同: %s", c1.ID)
	}
}
