package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOther_JSONLoading(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	configContent := `{
		"server": {"addr": "0.0.0.0", "port": 8080},
		"database": {"driver": "mysql", "dsn": "root@tcp(localhost)/db"},
		"jwt": {"secret": "my-secret", "expires": 3600},
		"cache": {"ttl": 300},
		"app_name": "my-app"
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := LoadJSONConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load JSON config: %v", err)
	}

	// Known fields should still parse correctly
	if config.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", config.Server.Port)
	}

	// Other should capture custom keys (jwt is now a known field, not in Other)
	if len(config.Other) != 2 {
		t.Errorf("Expected 2 custom keys in Other, got %d", len(config.Other))
	}

	// JWT should be parsed into Config.JWT directly
	if config.JWT.Secret != "my-secret" {
		t.Errorf("Expected JWT secret 'my-secret', got '%s'", config.JWT.Secret)
	}

	v, ok := config.GetOther("app_name")
	if !ok {
		t.Error("Expected 'app_name' key in Other")
	}
	if v != "my-app" {
		t.Errorf("Expected app_name='my-app', got %v", v)
	}
}

func TestOther_YAMLLoading(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `server:
  addr: "127.0.0.1"
  port: 9090
database:
  driver: "postgres"
  dsn: "postgres://localhost/db"
third_party:
  wechat:
    app_id: "wx123"
    secret: "abc"
email:
  smtp_host: "smtp.example.com"
  port: 587
`

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := LoadYAMLConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}

	// Known fields should still parse correctly
	if config.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", config.Server.Port)
	}
	if config.Database.Driver != "postgres" {
		t.Errorf("Expected driver 'postgres', got '%s'", config.Database.Driver)
	}

	// Other should capture custom keys
	if len(config.Other) != 2 {
		t.Errorf("Expected 2 custom keys in Other, got %d", len(config.Other))
	}

	if !config.HasOther("third_party") {
		t.Error("Expected 'third_party' key in Other")
	}

	if !config.HasOther("email") {
		t.Error("Expected 'email' key in Other")
	}
}

func TestOther_ToJSONIncludesOther(t *testing.T) {
	config := DefaultConfig()
	config.SetOther("app_name", "test-app")
	config.SetOther("custom_obj", map[string]interface{}{
		"key1": "val1",
		"key2": 123,
	})

	data, err := config.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if result["app_name"] != "test-app" {
		t.Errorf("Expected app_name='test-app', got %v", result["app_name"])
	}

	if _, ok := result["custom_obj"]; !ok {
		t.Error("Expected 'custom_obj' in JSON output")
	}

	if _, ok := result["server"]; !ok {
		t.Error("Expected 'server' in JSON output")
	}
}

func TestOther_ToYAMLIncludesOther(t *testing.T) {
	config := DefaultConfig()
	config.SetOther("app_name", "test-app")

	data, err := config.ToYAML()
	if err != nil {
		t.Fatalf("Failed to convert to YAML: %v", err)
	}

	if len(data) == 0 {
		t.Error("YAML data should not be empty")
	}

	yamlStr := string(data)
	if yamlStr == "" {
		t.Error("YAML string should not be empty")
	}
}

func TestOther_SetGetHasDelete(t *testing.T) {
	config := DefaultConfig()

	// SetOther
	config.SetOther("key1", "value1")
	config.SetOther("key2", 42)

	// HasOther
	if !config.HasOther("key1") {
		t.Error("Expected HasOther('key1') to be true")
	}

	// GetOther
	v, ok := config.GetOther("key1")
	if !ok || v != "value1" {
		t.Errorf("Expected GetOther('key1') = 'value1', got %v, %v", v, ok)
	}

	_, ok = config.GetOther("nonexistent")
	if ok {
		t.Error("Expected GetOther('nonexistent') to return false")
	}

	// DeleteOther
	config.DeleteOther("key1")
	if config.HasOther("key1") {
		t.Error("Expected key1 to be deleted")
	}
	if !config.HasOther("key2") {
		t.Error("Expected key2 to still exist")
	}
}

func TestOther_UnmarshalOther(t *testing.T) {
	config := DefaultConfig()

	type JWTConfig struct {
		Secret  string `json:"secret" yaml:"secret"`
		Expires int    `json:"expires" yaml:"expires"`
	}

	config.SetOther("jwt", map[string]interface{}{
		"secret":  "my-jwt-secret",
		"expires": 7200,
	})

	var jwtCfg JWTConfig
	err := config.UnmarshalOther("jwt", &jwtCfg)
	if err != nil {
		t.Fatalf("UnmarshalOther failed: %v", err)
	}

	if jwtCfg.Secret != "my-jwt-secret" {
		t.Errorf("Expected secret='my-jwt-secret', got '%s'", jwtCfg.Secret)
	}
	if jwtCfg.Expires != 7200 {
		t.Errorf("Expected expires=7200, got %d", jwtCfg.Expires)
	}

	// Test non-existent key
	err = config.UnmarshalOther("nonexistent", &jwtCfg)
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
}

func TestOther_AllOther(t *testing.T) {
	config := DefaultConfig()
	config.SetOther("a", 1)
	config.SetOther("b", 2)

	all := config.AllOther()
	if len(all) != 2 {
		t.Errorf("Expected 2 items, got %d", len(all))
	}

	// Should be a copy
	all["c"] = 3
	if config.HasOther("c") {
		t.Error("AllOther should return a copy, not reference")
	}
}

func TestOther_DefaultConfigHasEmptyOther(t *testing.T) {
	config := DefaultConfig()
	if config.Other == nil {
		t.Error("DefaultConfig().Other should be initialized (not nil)")
	}
	if len(config.Other) != 0 {
		t.Errorf("DefaultConfig().Other should be empty, got %d items", len(config.Other))
	}
}

func TestOther_NoCustomKeys(t *testing.T) {
	// When config file has only known keys, Other should be nil or empty
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	configContent := `{
		"server": {"addr": "0.0.0.0", "port": 8080},
		"database": {"driver": "mysql", "dsn": "root@tcp(localhost)/db"}
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := LoadJSONConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load JSON config: %v", err)
	}

	// Other should exist but be empty
	if config.Other == nil {
		t.Error("Other should not be nil even without custom keys")
	}
	if len(config.Other) != 0 {
		t.Errorf("Other should be empty, got %d items: %v", len(config.Other), config.Other)
	}

	// Empty Other should still marshal correctly
	data, err := config.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("JSON should not be empty")
	}
}
