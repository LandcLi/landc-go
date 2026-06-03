package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ApplyEnvOverrides 用环境变量覆盖配置
// 环境变量命名规则：LANDC_{SECTION}_{KEY}，例如：
//   - LANDC_SERVER_PORT=9090
//   - LANDC_DATABASE_DSN=root:pwd@tcp(localhost:3306)/db
//   - LANDC_REDIS_ADDR=localhost:6380
//   - LANDC_LOG_LEVEL=debug
func ApplyEnvOverrides(cfg *Config) {
	if cfg == nil {
		return
	}

	// Server
	if v := os.Getenv("LANDC_SERVER_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("LANDC_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("LANDC_SERVER_READ_TIMEOUT"); v != "" {
		if t, err := strconv.Atoi(v); err == nil {
			cfg.Server.ReadTimeout = t
		}
	}
	if v := os.Getenv("LANDC_SERVER_WRITE_TIMEOUT"); v != "" {
		if t, err := strconv.Atoi(v); err == nil {
			cfg.Server.WriteTimeout = t
		}
	}

	// Database
	if v := os.Getenv("LANDC_DATABASE_DRIVER"); v != "" {
		cfg.Database.Driver = v
	}
	if v := os.Getenv("LANDC_DATABASE_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("LANDC_DATABASE_MAX_OPEN_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.MaxOpenConns = n
		}
	}
	if v := os.Getenv("LANDC_DATABASE_MAX_IDLE_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.MaxIdleConns = n
		}
	}

	// Redis
	if v := os.Getenv("LANDC_REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("LANDC_REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("LANDC_REDIS_DB"); v != "" {
		if db, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = db
		}
	}

	// Log
	if v := os.Getenv("LANDC_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("LANDC_LOG_FORMAT"); v != "" {
		cfg.Log.Format = v
	}
	if v := os.Getenv("LANDC_LOG_OUTPUT"); v != "" {
		cfg.Log.Output = v
	}
}

// InitGlobalConfigWithPathAndEnv 从文件加载配置并应用环境变量覆盖
func InitGlobalConfigWithPathAndEnv(path string) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	if globalConfig != nil {
		return nil
	}

	config, err := LoadConfigFromFile(path)
	if err != nil {
		return fmt.Errorf("failed to load config from file: %w", err)
	}

	ApplyEnvOverrides(config)
	globalConfig = config
	return nil
}

// WatchConfig 监听配置文件变化（简单轮询实现）
// onChange 回调在配置文件发生变化时调用
func WatchConfig(path string, interval time.Duration, onChange func(*Config)) (stop func()) {
	done := make(chan struct{})
	var lastModTime time.Time

	if info, err := os.Stat(path); err == nil {
		lastModTime = info.ModTime()
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				info, err := os.Stat(path)
				if err != nil {
					continue
				}
				if info.ModTime().After(lastModTime) {
					lastModTime = info.ModTime()
					newConfig, err := LoadConfigFromFile(path)
					if err != nil {
						continue
					}
					ApplyEnvOverrides(newConfig)

					configMutex.Lock()
					globalConfig = newConfig
					configMutex.Unlock()

					if onChange != nil {
						onChange(newConfig)
					}
				}
			case <-done:
				return
			}
		}
	}()

	return func() { close(done) }
}

// GetEnv 获取环境变量，支持默认值
func GetEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// GetEnvInt 获取整数环境变量
func GetEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultValue
}

// GetEnvBool 获取布尔环境变量
func GetEnvBool(key string, defaultValue bool) bool {
	if v := os.Getenv(key); v != "" {
		v = strings.ToLower(v)
		return v == "true" || v == "1" || v == "yes"
	}
	return defaultValue
}
