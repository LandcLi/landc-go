package verifycode

import (
	"context"
	"testing"

	"github.com/LandcLi/landc-go/frame/pkg/cache"
)

func TestGenerateAndVerify(t *testing.T) {
	cache.InitGlobalCacheWithLocal(100)
	ctx := context.Background()

	code, err := Generate(ctx, "sms:13800000000")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("code length = %d, want 6", len(code))
	}

	if !Verify(ctx, "sms:13800000000", code) {
		t.Error("Verify should pass")
	}
	if Verify(ctx, "sms:13800000000", code) {
		t.Error("Verify should fail after one-time consumption")
	}
}

func TestSendInterval(t *testing.T) {
	cache.InitGlobalCacheWithLocal(100)
	ctx := context.Background()

	_, err := Generate(ctx, "k")
	if err != nil {
		t.Fatal(err)
	}
	// 立即再次生成：发送间隔未到 → 应拒绝（返回 tools 的 ErrIntervalLimited）
	if _, err := Generate(ctx, "k"); err == nil {
		t.Error("second Generate should be rate limited")
	}
}
