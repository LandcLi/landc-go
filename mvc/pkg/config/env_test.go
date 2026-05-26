package config

import (
	"os"
	"testing"
)

func TestApplyEnvOverrides(t *testing.T) {
	cfg := DefaultConfig()

	os.Setenv("LANDC_SERVER_PORT", "9090")
	os.Setenv("LANDC_DATABASE_DSN", "test-dsn")
	os.Setenv("LANDC_REDIS_ADDR", "redis:6380")
	os.Setenv("LANDC_LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("LANDC_SERVER_PORT")
		os.Unsetenv("LANDC_DATABASE_DSN")
		os.Unsetenv("LANDC_REDIS_ADDR")
		os.Unsetenv("LANDC_LOG_LEVEL")
	}()

	ApplyEnvOverrides(cfg)

	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Database.DSN != "test-dsn" {
		t.Errorf("Expected DSN 'test-dsn', got '%s'", cfg.Database.DSN)
	}
	if cfg.Redis.Addr != "redis:6380" {
		t.Errorf("Expected addr 'redis:6380', got '%s'", cfg.Redis.Addr)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Expected level 'debug', got '%s'", cfg.Log.Level)
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_KEY", "test_value")
	defer os.Unsetenv("TEST_KEY")

	if GetEnv("TEST_KEY", "default") != "test_value" {
		t.Error("GetEnv should return env value")
	}
	if GetEnv("NONEXIST_KEY", "default") != "default" {
		t.Error("GetEnv should return default for missing key")
	}
}

func TestGetEnvInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	if GetEnvInt("TEST_INT", 0) != 42 {
		t.Error("GetEnvInt should return parsed int")
	}
	if GetEnvInt("NONEXIST", 99) != 99 {
		t.Error("GetEnvInt should return default for missing key")
	}
}

func TestGetEnvBool(t *testing.T) {
	os.Setenv("TEST_BOOL", "true")
	defer os.Unsetenv("TEST_BOOL")

	if !GetEnvBool("TEST_BOOL", false) {
		t.Error("GetEnvBool should return true")
	}
	if GetEnvBool("NONEXIST", false) {
		t.Error("GetEnvBool should return default")
	}
}

func TestApplyEnvOverridesNil(t *testing.T) {
	// 不应 panic
	ApplyEnvOverrides(nil)
}
