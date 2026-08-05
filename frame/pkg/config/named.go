package config

import (
	"context"
	"fmt"
	"sync"

	"github.com/LandcLi/landc-go/frame/pkg/resource"
)

// namedConfigs 保存命名配置：name -> *Config。
var namedConfigs sync.Map

// InitNamedConfig 注册一份命名配置。
// 供库模式嵌入场景使用：宿主可为嵌入服务准备独立的命名配置。
func InitNamedConfig(name string, cfg *Config) error {
	if name == "" {
		return fmt.Errorf("config: named config name cannot be empty")
	}
	if cfg == nil {
		return fmt.Errorf("config: named config %q cannot be nil", name)
	}
	namedConfigs.Store(name, cfg)
	return nil
}

// HasNamedConfig 报告命名配置是否已注册。
func HasNamedConfig(name string) bool {
	_, ok := namedConfigs.Load(name)
	return ok
}

// GetNamedConfig 返回命名配置；未注册返回 nil。
func GetNamedConfig(name string) *Config {
	v, ok := namedConfigs.Load(name)
	if !ok {
		return nil
	}
	return v.(*Config)
}

// GetConfigFrom 从 ctx 解析命名配置。
// 作用域指定了 Config 且命名配置存在时返回命名配置；否则回退全局配置。
// 作用域指定的命名配置未注册时返回 nil（注册入口已校验，运行时 nil 属边界情况）。
func GetConfigFrom(ctx context.Context) *Config {
	if s, ok := resource.FromContext(ctx); ok && s.Config != "" {
		if cfg := GetNamedConfig(s.Config); cfg != nil {
			return cfg
		}
		return nil
	}
	return GetConfig()
}
