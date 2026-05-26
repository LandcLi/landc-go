package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type (
	Config struct {
		Server   ServerConfig           `json:"server" yaml:"server"`
		Database DatabaseConfig         `json:"database" yaml:"database"`
		Redis    RedisConfig            `json:"redis" yaml:"redis"`
		Log      LogConfig              `json:"log" yaml:"log"`
		Other    map[string]interface{} `json:"-" yaml:"-"`
	}

	ServerConfig struct {
		Addr             string `json:"addr" yaml:"addr"`
		Port             int    `json:"port" yaml:"port"`
		ReadTimeout      int    `json:"read_timeout" yaml:"read_timeout"`
		WriteTimeout     int    `json:"write_timeout" yaml:"write_timeout"`
		UseDefaultRoutes bool   `json:"use_default_routes" yaml:"use_default_routes"`
	}

	DatabaseConfig struct {
		Driver          string `json:"driver" yaml:"driver"`
		DSN             string `json:"dsn" yaml:"dsn"`
		MaxOpenConns    int    `json:"max_open_conns" yaml:"max_open_conns"`
		MaxIdleConns    int    `json:"max_idle_conns" yaml:"max_idle_conns"`
		ConnMaxLifetime int    `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	}

	RedisConfig struct {
		Addr     string `json:"addr" yaml:"addr"`
		Password string `json:"password" yaml:"password"`
		DB       int    `json:"db" yaml:"db"`
		PoolSize int    `json:"pool_size" yaml:"pool_size"`
	}

	LogConfig struct {
		Level      string `json:"level" yaml:"level"`
		Format     string `json:"format" yaml:"format"`
		Output     string `json:"output" yaml:"output"`
		MaxSize    int    `json:"max_size" yaml:"max_size"`
		MaxBackups int    `json:"max_backups" yaml:"max_backups"`
		MaxAge     int    `json:"max_age" yaml:"max_age"`
	}
)

var (
	globalConfig *Config
	configMutex  sync.RWMutex
)

func GetConfig() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return globalConfig
}

func InitGlobalConfigWithConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	globalConfig = config
	return nil
}

func InitGlobalConfigWithPath(path string) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	if globalConfig != nil {
		return nil
	}

	config, err := LoadConfigFromFile(path)
	if err != nil {
		return fmt.Errorf("failed to load config from file: %w", err)
	}

	globalConfig = config
	return nil
}

func InitGlobalConfigWithDefault() error {
	configMutex.Lock()
	defer configMutex.Unlock()

	if globalConfig != nil {
		return nil
	}

	globalConfig = DefaultConfig()
	return nil
}

func LoadConfigFromFile(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		return LoadJSONConfig(path)
	case ".yaml", ".yml":
		return LoadYAMLConfig(path)
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}
}

func LoadJSONConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}

	return &config, nil
}

func LoadYAMLConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return &config, nil
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Addr:             "0.0.0.0",
			Port:             8080,
			ReadTimeout:      60,
			WriteTimeout:     60,
			UseDefaultRoutes: true,
		},
		Database: DatabaseConfig{
			Driver:          "mysql",
			DSN:             "root:password@tcp(localhost:3306)/database?charset=utf8mb4&parseTime=True&loc=Local",
			MaxOpenConns:    100,
			MaxIdleConns:    10,
			ConnMaxLifetime: 3600,
		},
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
			PoolSize: 10,
		},
		Log: LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
		},
	}
}

func (c *Config) GetServerAddr() string {
	if c.Server.Addr == "" {
		return ":8080"
	}
	if c.Server.Port == 0 {
		return c.Server.Addr
	}
	return fmt.Sprintf("%s:%d", c.Server.Addr, c.Server.Port)
}

func (c *Config) Validate() error {
	if c.Server.Port < 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Driver == "" {
		return fmt.Errorf("database driver cannot be empty")
	}

	if c.Database.DSN == "" {
		return fmt.Errorf("database DSN cannot be empty")
	}

	return nil
}

func (c *Config) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

func (c *Config) ToYAML() ([]byte, error) {
	return yaml.Marshal(c)
}
