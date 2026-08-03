package workflow

import (
	frameconfig "github.com/LandcLi/landc-go/frame/pkg/config"
)

// Config 工作流框架业务配置
type Config struct {
	Engine    EngineConfig
	Scheduler SchedulerConfig
}

// EngineConfig 引擎配置
type EngineConfig struct {
	MaxParallelTasks int `json:"max_parallel_tasks"`
	DefaultTimeout   int `json:"default_timeout"` // 秒
	IdempotencyTTL   int `json:"idempotency_ttl"` // 秒
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	PollInterval int `json:"poll_interval"` // 秒
	BatchSize    int `json:"batch_size"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Engine: EngineConfig{
			MaxParallelTasks: 10,
			DefaultTimeout:   1800,
			IdempotencyTTL:   86400,
		},
		Scheduler: SchedulerConfig{
			PollInterval: 2,
			BatchSize:    20,
		},
	}
}

// readConfig 从框架配置中提取工作流配置
func readConfig() *Config {
	cfg := DefaultConfig()
	fc := frameconfig.GetConfig()
	if fc == nil || fc.Other == nil {
		return cfg
	}

	if v, ok := fc.Other["workflow"].(map[string]interface{}); ok {
		if e, ok := v["engine"].(map[string]interface{}); ok {
			if p, ok := e["max_parallel_tasks"].(int); ok {
				cfg.Engine.MaxParallelTasks = p
			}
			if t, ok := e["default_timeout"].(int); ok {
				cfg.Engine.DefaultTimeout = t
			}
			if t, ok := e["idempotency_ttl"].(int); ok {
				cfg.Engine.IdempotencyTTL = t
			}
		}
		if s, ok := v["scheduler"].(map[string]interface{}); ok {
			if p, ok := s["poll_interval"].(int); ok {
				cfg.Scheduler.PollInterval = p
			}
			if b, ok := s["batch_size"].(int); ok {
				cfg.Scheduler.BatchSize = b
			}
		}
	}
	return cfg
}
