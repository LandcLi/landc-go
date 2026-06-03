package auth

import (
	"testing"
	"time"
)

func TestGenerateAndParseToken(t *testing.T) {
	InitJWT(&JWTConfig{
		Secret:     "test-secret-key-123",
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
		Secret:     "test-secret-key-123",
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
		Secret:     "test-secret-key-123",
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
		Secret:     "test-secret-key-123",
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
		Secret:     "test-secret-key-123",
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
