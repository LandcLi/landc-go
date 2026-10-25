package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadYAMLConfigAppliesDefaults 验证配置文件中省略的字段会保留默认值。
//
// 回归背景：此前 LoadYAMLConfig 直接反序列化到零值 Config，导致 YAML 中未出现的
// 字段退化为 Go 零值。最具破坏性的是 server.use_default_routes 与
// server.health_check.enabled 变成 false——健康检查端点根本不会注册，
// k8s 探针与 Docker HEALTHCHECK 全部失效，而且现象是 401/404 混杂，极难定位。
func TestLoadYAMLConfigAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// 刻意只写 port，其余字段一律省略
	content := "server:\n  port: 9999\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadYAMLConfig(path)
	if err != nil {
		t.Fatalf("LoadYAMLConfig: %v", err)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("explicit port should win, got %d", cfg.Server.Port)
	}
	if !cfg.Server.UseDefaultRoutes {
		t.Error("server.use_default_routes should default to true when omitted")
	}
	if !cfg.Server.HealthCheck.Enabled {
		t.Error("server.health_check.enabled should default to true when omitted")
	}
	if cfg.Server.HealthCheck.LivenessPath != "/health" {
		t.Errorf("liveness path default lost, got %q", cfg.Server.HealthCheck.LivenessPath)
	}
	if cfg.Server.HealthCheck.ReadinessPath != "/ready" {
		t.Errorf("readiness path default lost, got %q", cfg.Server.HealthCheck.ReadinessPath)
	}
	if cfg.Server.ReadTimeout == 0 {
		t.Error("server.read_timeout default lost")
	}
	if cfg.Server.WriteTimeout == 0 {
		t.Error("server.write_timeout default lost")
	}
}

// TestLoadYAMLConfigExplicitDisableWins 验证显式关闭能覆盖默认值。
// 默认值补全不能反过来把用户明确写的 false 改成 true。
func TestLoadYAMLConfigExplicitDisableWins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := "server:\n  use_default_routes: false\n  health_check:\n    enabled: false\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadYAMLConfig(path)
	if err != nil {
		t.Fatalf("LoadYAMLConfig: %v", err)
	}

	if cfg.Server.UseDefaultRoutes {
		t.Error("explicit false for use_default_routes must be respected")
	}
	if cfg.Server.HealthCheck.Enabled {
		t.Error("explicit false for health_check.enabled must be respected")
	}
}

// TestLoadJSONConfigAppliesDefaults JSON 通道行为应与 YAML 一致。
func TestLoadJSONConfigAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	content := `{"server":{"port":7777}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadJSONConfig(path)
	if err != nil {
		t.Fatalf("LoadJSONConfig: %v", err)
	}

	if cfg.Server.Port != 7777 {
		t.Errorf("explicit port should win, got %d", cfg.Server.Port)
	}
	if !cfg.Server.UseDefaultRoutes {
		t.Error("server.use_default_routes should default to true when omitted")
	}
	if !cfg.Server.HealthCheck.Enabled {
		t.Error("server.health_check.enabled should default to true when omitted")
	}
}
