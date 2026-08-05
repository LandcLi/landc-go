package config

import (
	"context"
	"testing"

	"github.com/LandcLi/landc-go/frame/pkg/resource"
)

func TestNamedConfig(t *testing.T) {
	cfg := DefaultConfig()
	if err := InitNamedConfig("lum", cfg); err != nil {
		t.Fatalf("InitNamedConfig: %v", err)
	}
	if !HasNamedConfig("lum") || GetNamedConfig("lum") != cfg {
		t.Error("named config not retrievable")
	}
	if HasNamedConfig("nope") {
		t.Error("HasNamedConfig(nope) should be false")
	}
	if err := InitNamedConfig("", cfg); err == nil {
		t.Error("empty name should error")
	}
}

func TestGetConfigFrom(t *testing.T) {
	named := DefaultConfig()
	named.Server.Addr = "lum-addr"
	_ = InitNamedConfig("lum", named)

	global := DefaultConfig()
	global.Server.Addr = "global-addr"
	_ = InitGlobalConfigWithConfig(global)

	ctx := resource.WithScope(context.Background(), resource.Scope{Name: "lum", Config: "lum"})
	if got := GetConfigFrom(ctx); got == nil || got.Server.Addr != "lum-addr" {
		t.Errorf("GetConfigFrom(scoped) = %+v, want named config", got)
	}

	if got := GetConfigFrom(context.Background()); got == nil || got.Server.Addr != "global-addr" {
		t.Errorf("GetConfigFrom(plain) = %+v, want global config", got)
	}

	missing := resource.WithScope(context.Background(), resource.Scope{Name: "x", Config: "nope"})
	if got := GetConfigFrom(missing); got != nil {
		t.Errorf("GetConfigFrom(unregistered) = %+v, want nil (no silent fallback)", got)
	}
}
