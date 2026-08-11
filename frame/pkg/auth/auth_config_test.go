package auth

import (
	"testing"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/config"
)

func resetJWTConfig() {
	InitJWT(nil)
}

func TestInitFromConfig_NoJWTSection(t *testing.T) {
	resetJWTConfig()
	InitFromConfig(&config.Config{})
	if GetJWTConfig() != nil {
		t.Error("expected no-op when jwt section empty")
	}
}

func TestInitFromConfig_OnlyIssuer(t *testing.T) {
	resetJWTConfig()
	InitFromConfig(&config.Config{JWT: config.JWTConfig{Issuer: "myapp"}})
	if GetJWTConfig() != nil {
		t.Error("expected no-op when only issuer configured (no key field)")
	}
}

func TestInitFromConfig_SecretWithoutExpire(t *testing.T) {
	resetJWTConfig()
	InitFromConfig(&config.Config{JWT: config.JWTConfig{
		Secret: "0123456789abcdef0123456789abcdef",
		Issuer: "myapp",
	}})

	got := GetJWTConfig()
	if got == nil {
		t.Fatal("expected jwt initialized with secret only")
	}
	if got.Secret != "0123456789abcdef0123456789abcdef" || got.Issuer != "myapp" {
		t.Errorf("config mismatch: %+v", got)
	}
	if got.ExpireTime != 0 {
		t.Errorf("expected zero expire (not configured), got %v", got.ExpireTime)
	}
}

func TestInitFromConfig_WithExpire(t *testing.T) {
	resetJWTConfig()
	InitFromConfig(&config.Config{JWT: config.JWTConfig{
		Secret:     "0123456789abcdef0123456789abcdef",
		ExpireTime: "2h",
		Issuer:     "myapp",
	}})

	got := GetJWTConfig()
	if got == nil || got.ExpireTime != 2*time.Hour {
		t.Fatalf("expected expire 2h, got %+v", got)
	}

	token, err := GenerateToken(1, "landc")
	if err != nil {
		t.Fatalf("generate token failed: %v", err)
	}
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("parse token failed: %v", err)
	}
	if claims.UserID != 1 || claims.Username != "landc" {
		t.Errorf("claims mismatch: %+v", claims)
	}
}

func TestGenerateToken_NotInitializedHint(t *testing.T) {
	resetJWTConfig()
	_, err := GenerateToken(1, "landc")
	if err == nil {
		t.Fatal("expected error when jwt not initialized")
	}
	if err.Error() != "JWT config not initialized: call auth.InitJWT() (or use landc bootstrap / web.NewServer which auto-init from config.yaml jwt section)" {
		t.Errorf("expected init hint in error, got: %v", err)
	}
}
