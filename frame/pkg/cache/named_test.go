package cache

import (
	"context"
	"testing"

	"github.com/LandcLi/landc-go/frame/pkg/resource"
)

func TestNamedCache(t *testing.T) {
	InitNamedCacheWithLocal("lum", 100)
	if !HasNamedCache("lum") || GetNamedCache("lum") == nil {
		t.Error("named cache not retrievable")
	}
	if HasNamedCache("nope") {
		t.Error("HasNamedCache(nope) should be false")
	}
	if err := InitNamedCacheWithConfig("", nil); err == nil {
		t.Error("empty name should error")
	}
	if err := InitNamedCacheWithConfig("err", nil); err == nil {
		t.Error("nil config should error")
	}
}

func TestGetCacheFrom(t *testing.T) {
	InitNamedCacheWithLocal("lum", 100)
	InitGlobalCacheWithLocal(1000)

	ctx := resource.WithScope(context.Background(), resource.Scope{Name: "lum", Cache: "lum"})
	if GetCacheFrom(ctx) == nil {
		t.Error("GetCacheFrom(scoped) should return named cache")
	}

	scoped := resource.WithScope(context.Background(), resource.Scope{Name: "lum", Cache: "lum"})
	if GetCacheFrom(scoped) == GetCache() {
		t.Error("GetCacheFrom(scoped) should NOT be the global cache")
	}

	if GetCacheFrom(context.Background()) != GetCache() {
		t.Error("GetCacheFrom(plain) should fall back to global cache")
	}
}
