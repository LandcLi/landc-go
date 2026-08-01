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
		Server   ServerConfig   `json:"server" yaml:"server"`
		Database DatabaseConfig `json:"database" yaml:"database"`
		Redis    RedisConfig    `json:"redis" yaml:"redis"`
		Log      LogConfig      `json:"log" yaml:"log"`
		JWT      JWTConfig      `json:"jwt" yaml:"jwt"`
		// Other 存储用户自定义配置项（非框架标准字段）。
		// 从 JSON/YAML 加载时，所有不在 server/database/redis/log/jwt 下的顶级键
		// 都会被自动捕获到此处。序列化时也会合并回顶层输出。
		Other map[string]interface{} `json:"-" yaml:"-"`
	}

	JWTConfig struct {
		Secret     string `json:"secret" yaml:"secret"`
		ExpireTime string `json:"expire_time" yaml:"expire_time"`
		Issuer     string `json:"issuer" yaml:"issuer"`
	}

	ServerConfig struct {
		Addr             string            `json:"addr" yaml:"addr"`
		Port             int               `json:"port" yaml:"port"`
		ReadTimeout      int               `json:"read_timeout" yaml:"read_timeout"`
		WriteTimeout     int               `json:"write_timeout" yaml:"write_timeout"`
		UseDefaultRoutes bool              `json:"use_default_routes" yaml:"use_default_routes"`
		HealthCheck      HealthCheckConfig `json:"health_check" yaml:"health_check"`
		RequestTimeout   int               `json:"request_timeout" yaml:"request_timeout"`
	}

	HealthCheckConfig struct {
		Enabled       bool   `json:"enabled" yaml:"enabled"`
		LivenessPath  string `json:"liveness_path" yaml:"liveness_path"`
		ReadinessPath string `json:"readiness_path" yaml:"readiness_path"`
		StartupPath   string `json:"startup_path,omitempty" yaml:"startup_path,omitempty"`
		DatabaseCheck bool   `json:"database_check" yaml:"database_check"`
		RedisCheck    bool   `json:"redis_check" yaml:"redis_check"`
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
	// 生产环境安全默认值：密钥优先从环境变量注入，避免硬编码
	// 若环境变量也未设置，则留空并由 auth 层拒绝签发/解析 Token
	jwtSecret := os.Getenv("LANDC_JWT_SECRET")

	dsn := os.Getenv("LANDC_DB_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(localhost:3306)/database?charset=utf8mb4&parseTime=True&loc=Local"
	}

	return &Config{
		Server: ServerConfig{
			Addr:             "0.0.0.0",
			Port:             8080,
			ReadTimeout:      60,
			WriteTimeout:     60,
			UseDefaultRoutes: true,
			HealthCheck: HealthCheckConfig{
				Enabled:       true,
				LivenessPath:  "/health",
				ReadinessPath: "/ready",
				StartupPath:   "",
				DatabaseCheck: true,
				RedisCheck:    false,
			},
			RequestTimeout: 0,
		},
		Database: DatabaseConfig{
			Driver:          "mysql",
			DSN:             dsn,
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
		JWT: JWTConfig{
			Secret:     jwtSecret,
			ExpireTime: "2h",
			Issuer:     "app",
		},
		Other: make(map[string]interface{}),
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

// ============================================================================
// 自定义 JSON / YAML 序列化与反序列化
// 使得 Other 中存储的用户自定义配置项能够：
//   1. 从文件中自动捕获（反序列化时未知键 → Other）
//   2. 序列化时自动合并回顶层输出（Other 键 → 顶层键）
// ============================================================================

// UnmarshalJSON 自定义 JSON 反序列化，将未知顶级字段捕获到 Other 中。
func (c *Config) UnmarshalJSON(data []byte) error {
	// 第一步：反序列化已知字段（server/database/redis/log）
	type Alias Config
	if err := json.Unmarshal(data, (*Alias)(c)); err != nil {
		return err
	}

	// 第二步：解析所有顶级键，提取未知字段
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// 移除框架标准字段
	delete(raw, "server")
	delete(raw, "database")
	delete(raw, "redis")
	delete(raw, "log")
	delete(raw, "jwt")

	// 剩余字段存入 Other
	c.Other = make(map[string]interface{})
	for k, v := range raw {
		var val interface{}
		if err := json.Unmarshal(v, &val); err != nil {
			return fmt.Errorf("failed to unmarshal other key %q: %w", k, err)
		}
		c.Other[k] = val
	}
	return nil
}

// MarshalJSON 自定义 JSON 序列化，将 Other 中的字段合并到顶层输出。
func (c *Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	data, err := json.Marshal((*Alias)(c))
	if err != nil {
		return nil, err
	}

	if len(c.Other) == 0 {
		return data, nil
	}

	// 合并 Other 字段到顶层
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	for k, v := range c.Other {
		result[k] = v
	}
	return json.Marshal(result)
}

// UnmarshalYAML 自定义 YAML 反序列化，将未知顶级字段捕获到 Other 中。
func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	// 第一步：反序列化已知字段
	type Alias Config
	if err := value.Decode((*Alias)(c)); err != nil {
		return err
	}

	// 第二步：从 YAML 节点中提取未知顶级字段
	knownKeys := map[string]bool{
		"server": true, "database": true, "redis": true, "log": true, "jwt": true,
	}

	if value.Kind == yaml.MappingNode {
		c.Other = make(map[string]interface{})
		for i := 0; i < len(value.Content); i += 2 {
			key := value.Content[i].Value
			if !knownKeys[key] {
				var val interface{}
				if err := value.Content[i+1].Decode(&val); err != nil {
					return fmt.Errorf("failed to decode other key %q: %w", key, err)
				}
				c.Other[key] = val
			}
		}
	} else {
		c.Other = make(map[string]interface{})
	}
	return nil
}

// MarshalYAML 自定义 YAML 序列化，将 Other 中的字段合并到顶层输出。
func (c *Config) MarshalYAML() (interface{}, error) {
	type Alias Config
	data, err := yaml.Marshal((*Alias)(c))
	if err != nil {
		return nil, err
	}

	if len(c.Other) == 0 {
		var result map[string]interface{}
		yaml.Unmarshal(data, &result)
		return result, nil
	}

	// 合并 Other 字段到顶层
	var result map[string]interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	for k, v := range c.Other {
		result[k] = v
	}
	return result, nil
}

// ============================================================================
// Other 字段访问方法
// ============================================================================

// GetOther 获取 Other 中指定 key 的值，返回值和是否存在。
func (c *Config) GetOther(key string) (interface{}, bool) {
	if c.Other == nil {
		return nil, false
	}
	v, ok := c.Other[key]
	return v, ok
}

// SetOther 设置 Other 中指定 key 的值。
func (c *Config) SetOther(key string, value interface{}) {
	if c.Other == nil {
		c.Other = make(map[string]interface{})
	}
	c.Other[key] = value
}

// HasOther 检查 Other 中是否存在指定 key。
func (c *Config) HasOther(key string) bool {
	_, ok := c.Other[key]
	return ok
}

// DeleteOther 删除 Other 中指定 key。
func (c *Config) DeleteOther(key string) {
	delete(c.Other, key)
}

// UnmarshalOther 将 Other 中指定 key 的值反序列化到 target 中。
// target 必须是指针类型。适用于将 map[string]interface{} 转为类型化结构体。
//
// 用法示例：
//
//	var jwtCfg JWTConfig
//	err := cfg.UnmarshalOther("jwt", &jwtCfg)
func (c *Config) UnmarshalOther(key string, target interface{}) error {
	val, ok := c.GetOther(key)
	if !ok {
		return fmt.Errorf("other key %q not found", key)
	}

	// 通过 JSON 中转实现 map[string]interface{} → 类型化结构体
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal other value for key %q: %w", key, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal other key %q: %w", key, err)
	}
	return nil
}

// AllOther 返回 Other 的浅拷贝副本，方便遍历所有自定义配置项。
func (c *Config) AllOther() map[string]interface{} {
	if c.Other == nil {
		return nil
	}
	result := make(map[string]interface{}, len(c.Other))
	for k, v := range c.Other {
		result[k] = v
	}
	return result
}
