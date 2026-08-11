package web

import (
	"testing"

	"github.com/LandcLi/landc-go/frame/pkg/auth"
	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/gin-gonic/gin"
)

// TestNewServer_AutoInitJWT 验证 web.NewServer 直接启动链路也会自动初始化 JWT
// （此前只有 bootstrap/cmd 链路初始化，web 直接启动时 middleware.Auth 会报
// "JWT config not initialized"）。
func TestNewServer_AutoInitJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.InitJWT(nil) // 复位全局 JWT

	// 模拟用户手动加载了含 jwt 段的全局配置（web.NewServer 直接启动的场景）
	if err := config.InitGlobalConfigWithConfig(&config.Config{
		JWT: config.JWTConfig{
			Secret:     "0123456789abcdef0123456789abcdef",
			ExpireTime: "1h",
			Issuer:     "test-issuer",
		},
	}); err != nil {
		t.Fatalf("init global config: %v", err)
	}

	_ = NewServer(nil) // web.NewServer 直接启动

	got := auth.GetJWTConfig()
	if got == nil {
		t.Fatal("expected JWT auto-initialized by web.NewServer")
	}
	if got.Secret != "0123456789abcdef0123456789abcdef" || got.Issuer != "test-issuer" {
		t.Errorf("JWT config mismatch: %+v", got)
	}
}
