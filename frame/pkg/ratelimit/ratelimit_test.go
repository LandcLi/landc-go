package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/cache"
)

func TestAllowInterval(t *testing.T) {
	cache.InitGlobalCacheWithLocal(100)
	ctx := context.Background()

	if !AllowInterval(ctx, "sms:13800000000", 60*time.Second) {
		t.Error("first allow should pass")
	}
	if AllowInterval(ctx, "sms:13800000000", 60*time.Second) {
		t.Error("within window should be rejected")
	}
	if !AllowInterval(ctx, "sms:13900000000", 60*time.Second) {
		t.Error("different key should pass")
	}
}

func TestAllowCount(t *testing.T) {
	cache.InitGlobalCacheWithLocal(100)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if !AllowCount(ctx, "daily:u1", 3, time.Hour) {
			t.Fatalf("allow %d should pass", i)
		}
	}
	if AllowCount(ctx, "daily:u1", 3, time.Hour) {
		t.Error("4th allow should be rejected")
	}
}

func TestAllowCountConcurrent(t *testing.T) {
	cache.InitGlobalCacheWithLocal(100)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = AllowCount(ctx, "conc", 100000, time.Hour)
			}
		}()
	}
	wg.Wait()

	// 原子自增：计数必须精确为 400
	v, _ := cache.GetCache().Get(context.Background(), "conc")
	if v != "400" {
		t.Errorf("counter = %s, want 400", v)
	}
}
