package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

var (
	testConfigMutex sync.Mutex
)

func resetConfig() {
	testConfigMutex.Lock()
	defer testConfigMutex.Unlock()
	globalConfig = nil
}

func TestDefaultConfig(t *testing.T) {
	resetConfig()
	config := DefaultConfig()

	if config.Server.Addr != "0.0.0.0" {
		t.Errorf("Expected server addr '0.0.0.0', got '%s'", config.Server.Addr)
	}

	if config.Server.Port != 8080 {
		t.Errorf("Expected server port 8080, got %d", config.Server.Port)
	}

	if config.Database.Driver != "mysql" {
		t.Errorf("Expected database driver 'mysql', got '%s'", config.Database.Driver)
	}

	if config.Log.Level != "info" {
		t.Errorf("Expected log level 'info', got '%s'", config.Log.Level)
	}
}

func TestInitGlobalConfigWithConfig(t *testing.T) {
	resetConfig()
	config := DefaultConfig()
	config.Server.Port = 9000

	err := InitGlobalConfigWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}

	retrieved := GetConfig()
	if retrieved.Server.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", retrieved.Server.Port)
	}

	err = InitGlobalConfigWithConfig(nil)
	if err == nil {
		t.Error("Should return error for nil config")
	}
}

func TestInitGlobalConfigWithDefault(t *testing.T) {
	resetConfig()
	err := InitGlobalConfigWithDefault()
	if err != nil {
		t.Fatalf("Failed to init default config: %v", err)
	}

	config := GetConfig()
	if config == nil {
		t.Fatal("Config should not be nil")
	}

	if config.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", config.Server.Port)
	}
}

func TestInitGlobalConfigWithPath(t *testing.T) {
	resetConfig()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	configContent := `{
		"server": {
			"addr": "127.0.0.1",
			"port": 8080,
			"use_default_routes": false
		},
		"database": {
			"driver": "mysql",
			"dsn": "test:password@tcp(localhost:3306)/test?charset=utf8mb4"
		}
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	err = InitGlobalConfigWithPath(configPath)
	if err != nil {
		t.Fatalf("Failed to init config with path: %v", err)
	}

	config := GetConfig()
	if config.Server.Addr != "127.0.0.1" {
		t.Errorf("Expected addr '127.0.0.1', got '%s'", config.Server.Addr)
	}

	if config.Server.UseDefaultRoutes {
		t.Error("Expected use_default_routes to be false")
	}
}

func TestInitGlobalConfigWithPath_NotOverwrite(t *testing.T) {
	resetConfig()
	err := InitGlobalConfigWithDefault()
	if err != nil {
		t.Fatalf("Failed to init default config: %v", err)
	}

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	configContent := `{
		"server": {
			"port": 9999
		}
	}`

	err = os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	err = InitGlobalConfigWithPath(configPath)
	if err != nil {
		t.Fatalf("Failed to init config with path: %v", err)
	}

	config := GetConfig()
	if config.Server.Port != 8080 {
		t.Errorf("Expected port 8080 (not overwritten), got %d", config.Server.Port)
	}
}

func TestLoadJSONConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	configContent := `{
		"server": {
			"addr": "192.168.1.1",
			"port": 9090
		},
		"database": {
			"driver": "postgres",
			"dsn": "postgres://user:pass@localhost/db"
		}
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := LoadJSONConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load JSON config: %v", err)
	}

	if config.Server.Addr != "192.168.1.1" {
		t.Errorf("Expected addr '192.168.1.1', got '%s'", config.Server.Addr)
	}

	if config.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", config.Server.Port)
	}

	if config.Database.Driver != "postgres" {
		t.Errorf("Expected driver 'postgres', got '%s'", config.Database.Driver)
	}
}

func TestLoadYAMLConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `server:
  addr: "10.0.0.1"
  port: 7070
database:
  driver: "sqlite"
  dsn: "file:test.db"
`

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := LoadYAMLConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}

	if config.Server.Addr != "10.0.0.1" {
		t.Errorf("Expected addr '10.0.0.1', got '%s'", config.Server.Addr)
	}

	if config.Server.Port != 7070 {
		t.Errorf("Expected port 7070, got %d", config.Server.Port)
	}

	if config.Database.Driver != "sqlite" {
		t.Errorf("Expected driver 'sqlite', got '%s'", config.Database.Driver)
	}
}

func TestLoadConfigFromFile_UnsupportedFormat(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.xml")

	err := os.WriteFile(configPath, []byte("<config></config>"), 0o600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err = LoadConfigFromFile(configPath)
	if err == nil {
		t.Error("Should return error for unsupported format")
	}
}

func TestLoadConfigFromFile_NotFound(t *testing.T) {
	_, err := LoadConfigFromFile("/nonexistent/config.json")
	if err == nil {
		t.Error("Should return error for nonexistent file")
	}
}

func TestGetServerAddr(t *testing.T) {
	config := DefaultConfig()

	addr := config.GetServerAddr()
	if addr != "0.0.0.0:8080" {
		t.Errorf("Expected '0.0.0.0:8080', got '%s'", addr)
	}

	config.Server.Port = 0
	addr = config.GetServerAddr()
	if addr != "0.0.0.0" {
		t.Errorf("Expected '0.0.0.0', got '%s'", addr)
	}

	config.Server.Addr = ""
	addr = config.GetServerAddr()
	if addr != ":8080" {
		t.Errorf("Expected ':8080', got '%s'", addr)
	}
}

func TestValidate(t *testing.T) {
	config := DefaultConfig()

	err := config.Validate()
	if err != nil {
		t.Errorf("Default config should be valid: %v", err)
	}

	config.Server.Port = -1
	err = config.Validate()
	if err == nil {
		t.Error("Should return error for invalid port")
	}

	config.Server.Port = 70000
	err = config.Validate()
	if err == nil {
		t.Error("Should return error for port > 65535")
	}

	config.Server.Port = 8080
	config.Database.Driver = ""
	err = config.Validate()
	if err == nil {
		t.Error("Should return error for empty database driver")
	}

	config.Database.Driver = "mysql"
	config.Database.DSN = ""
	err = config.Validate()
	if err == nil {
		t.Error("Should return error for empty database DSN")
	}
}

func TestToJSON(t *testing.T) {
	config := DefaultConfig()

	data, err := config.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	if len(data) == 0 {
		t.Error("JSON data should not be empty")
	}

	if string(data)[0] != '{' {
		t.Error("JSON should start with '{'")
	}
}

func TestToYAML(t *testing.T) {
	config := DefaultConfig()

	data, err := config.ToYAML()
	if err != nil {
		t.Fatalf("Failed to convert to YAML: %v", err)
	}

	if len(data) == 0 {
		t.Error("YAML data should not be empty")
	}
}
