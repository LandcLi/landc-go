package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetOtherPath 覆盖嵌套读取逻辑本身。
func TestGetOtherPath(t *testing.T) {
	cfg := &Config{
		Other: map[string]interface{}{
			"server": map[string]interface{}{
				"gin_mode":    "release",
				"enable_cors": true,
				"port":        1234,
			},
			"org_service": map[string]interface{}{
				"base_url": "http://127.0.0.1:8083",
			},
			"flat_key": "flat_value",
		},
	}

	t.Run("nested string", func(t *testing.T) {
		if got := cfg.GetOtherString("server", "gin_mode"); got != "release" {
			t.Errorf("GetOtherString = %q, want release", got)
		}
		if got := cfg.GetOtherString("org_service", "base_url"); got != "http://127.0.0.1:8083" {
			t.Errorf("GetOtherString = %q", got)
		}
	})

	t.Run("nested bool", func(t *testing.T) {
		v, ok := cfg.GetOtherBool("server", "enable_cors")
		if !ok || !v {
			t.Errorf("GetOtherBool = (%v, %v), want (true, true)", v, ok)
		}
	})

	t.Run("top level still works", func(t *testing.T) {
		if got := cfg.GetOtherString("flat_key"); got != "flat_value" {
			t.Errorf("top-level GetOtherString = %q", got)
		}
	})

	t.Run("missing path returns zero values", func(t *testing.T) {
		if got := cfg.GetOtherString("server", "nope"); got != "" {
			t.Errorf("missing key = %q, want empty", got)
		}
		if v, ok := cfg.GetOtherBool("nope", "deeper"); ok || v {
			t.Errorf("missing bool = (%v, %v), want (false, false)", v, ok)
		}
	})

	t.Run("type mismatch is ignored", func(t *testing.T) {
		if got := cfg.GetOtherString("server", "port"); got != "" {
			t.Errorf("type mismatch string = %q, want empty", got)
		}
		if v, ok := cfg.GetOtherBool("server", "port"); ok || v {
			t.Errorf("type mismatch bool = (%v, %v), want (false, false)", v, ok)
		}
		if got := cfg.GetOtherString("flat_key", "child"); got != "" {
			t.Errorf("non-map intermediate = %q, want empty", got)
		}
	})

	t.Run("nil receiver is safe", func(t *testing.T) {
		var nilCfg *Config
		if _, ok := nilCfg.GetOtherPath("server"); ok {
			t.Error("nil receiver should not report hit")
		}
	})

	t.Run("yaml v2 style map is handled", func(t *testing.T) {
		v2 := &Config{Other: map[string]interface{}{
			"server": map[interface{}]interface{}{"gin_mode": "release"},
		}}
		if got := v2.GetOtherString("server", "gin_mode"); got != "release" {
			t.Errorf("map[interface{}]interface{} lookup = %q, want release", got)
		}
	})
}

// TestLoadYAMLConfig_FlattensOtherSection 用**真实 YAML 解析**验证 other 段提升。
//
// 这是本项目最隐蔽的一类缺陷：Other 的语义是「未声明的顶层键」，
// 而历史文档让用户把业务配置写在 other: 段下，解析结果是
// Other["other"] = {...}，于是 GetOther("gin_mode") 永远取不到值 ——
// 配置写了不生效，且没有任何报错。用合成 map 写的测试发现不了这个问题，
// 必须走真实的 LoadYAMLConfig。
func TestLoadYAMLConfig_FlattensOtherSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := `
server:
  addr: "0.0.0.0"
  port: 8080
other:
  gin_mode: "release"
  enable_cors: true
  org_service:
    base_url: "http://127.0.0.1:8083"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadYAMLConfig(path)
	if err != nil {
		t.Fatalf("LoadYAMLConfig: %v", err)
	}

	if got := cfg.GetOtherString("gin_mode"); got != "release" {
		t.Errorf("GetOtherString(gin_mode) = %q, want release", got)
	}
	if v, ok := cfg.GetOtherBool("enable_cors"); !ok || !v {
		t.Errorf("GetOtherBool(enable_cors) = (%v, %v), want (true, true)", v, ok)
	}
	if got := cfg.GetOtherString("org_service", "base_url"); got != "http://127.0.0.1:8083" {
		t.Errorf("GetOtherString(org_service.base_url) = %q", got)
	}
	// 原始结构保留，不影响回写
	if _, ok := cfg.Other["other"]; !ok {
		t.Error("原始 other 段应保留，便于原样回写")
	}
}

// TestLoadYAMLConfig_TopLevelBusinessKeys 验证「业务配置直接写顶层」这条正路。
func TestLoadYAMLConfig_TopLevelBusinessKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := `
server:
  port: 8080
org_service:
  base_url: "http://org:8083"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadYAMLConfig(path)
	if err != nil {
		t.Fatalf("LoadYAMLConfig: %v", err)
	}
	if got := cfg.GetOtherString("org_service", "base_url"); got != "http://org:8083" {
		t.Errorf("顶层业务配置读取失败: %q", got)
	}
}

// TestLoadYAMLConfig_TopLevelWinsOverOtherSection 顶层同名键优先，避免两处冲突时行为不确定。
func TestLoadYAMLConfig_TopLevelWinsOverOtherSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := `
server:
  port: 8080
gin_mode: "top"
other:
  gin_mode: "nested"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadYAMLConfig(path)
	if err != nil {
		t.Fatalf("LoadYAMLConfig: %v", err)
	}
	if got := cfg.GetOtherString("gin_mode"); got != "top" {
		t.Errorf("顶层应优先，实际 %q", got)
	}
}

// TestLoadJSONConfig_FlattensOtherSection JSON 路径同样支持 other 段。
func TestLoadJSONConfig_FlattensOtherSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	content := `{"server":{"port":8080},"other":{"gin_mode":"release","org_service":{"base_url":"http://org:8083"}}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadJSONConfig(path)
	if err != nil {
		t.Fatalf("LoadJSONConfig: %v", err)
	}
	if got := cfg.GetOtherString("gin_mode"); got != "release" {
		t.Errorf("GetOtherString(gin_mode) = %q, want release", got)
	}
	if got := cfg.GetOtherString("org_service", "base_url"); got != "http://org:8083" {
		t.Errorf("GetOtherString(org_service.base_url) = %q", got)
	}
}
