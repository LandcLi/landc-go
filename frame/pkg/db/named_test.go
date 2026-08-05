package db

import (
	"context"
	"testing"

	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/LandcLi/landc-go/frame/pkg/resource"
)

func TestNamedDBRegistry(t *testing.T) {
	if HasNamedDB("nope") {
		t.Error("HasNamedDB(nope) should be false")
	}
	if GetNamedDB("nope") != nil {
		t.Error("GetNamedDB(nope) should be nil")
	}
	if err := InitNamedDB("", nil); err == nil {
		t.Error("empty name should error")
	}
	if err := InitNamedDB("err", nil); err == nil {
		t.Error("nil config should error")
	}
}

func TestGetDBFromFallback(t *testing.T) {
	// 无作用域 → 回退全局（未初始化时为 nil）
	if got := GetDBFrom(context.Background()); got != globalDB {
		t.Error("GetDBFrom(plain) should fall back to global db")
	}

	// 作用域指定未注册命名 → 返回 nil（不静默回退全局）
	scoped := resource.WithScope(context.Background(), resource.Scope{Name: "x", DB: "nope"})
	if got := GetDBFrom(scoped); got != nil {
		t.Error("GetDBFrom(unregistered) should return nil, not silently fall back")
	}
}

var _ = config.DatabaseConfig{}
