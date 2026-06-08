package bootstrap

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/auth"
	"github.com/LandcLi/landc-go/frame/pkg/cache"
	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/LandcLi/landc-go/frame/pkg/db"
	"github.com/LandcLi/landc-go/frame/pkg/trace"
)

type (
	Bootstrap struct {
		configLoader ConfigLoader
		configPath   string
		components   []Component
		beforeInit   []InitFunc
		afterInit    []InitFunc
		beforeRun    []RunFunc
		afterRun     []RunFunc
		autoInit     bool
	}

	ConfigLoader interface {
		Load(path string) error
	}

	Component interface {
		Name() string
		Init(ctx context.Context) error
		Close() error
	}

	InitFunc func(ctx context.Context) error
	RunFunc  func(ctx context.Context) error
)

func New() *Bootstrap {
	return &Bootstrap{
		components: make([]Component, 0),
		beforeInit: make([]InitFunc, 0),
		afterInit:  make([]InitFunc, 0),
		beforeRun:  make([]RunFunc, 0),
		afterRun:   make([]RunFunc, 0),
		autoInit:   true,
	}
}

func (b *Bootstrap) SetConfigLoader(loader ConfigLoader) *Bootstrap {
	b.configLoader = loader
	return b
}

func (b *Bootstrap) SetConfigPath(path string) *Bootstrap {
	b.configPath = path
	return b
}

func (b *Bootstrap) AddComponent(components ...Component) *Bootstrap {
	b.components = append(b.components, components...)
	return b
}

func (b *Bootstrap) BeforeInit(fn InitFunc) *Bootstrap {
	b.beforeInit = append(b.beforeInit, fn)
	return b
}

func (b *Bootstrap) AfterInit(fn InitFunc) *Bootstrap {
	b.afterInit = append(b.afterInit, fn)
	return b
}

func (b *Bootstrap) BeforeRun(fn RunFunc) *Bootstrap {
	b.beforeRun = append(b.beforeRun, fn)
	return b
}

func (b *Bootstrap) AfterRun(fn RunFunc) *Bootstrap {
	b.afterRun = append(b.afterRun, fn)
	return b
}

func (b *Bootstrap) EnableAutoInit(enable bool) *Bootstrap {
	b.autoInit = enable
	return b
}

func (b *Bootstrap) Init(ctx context.Context) error {
	ctx = trace.InitTrace(ctx)

	if b.configLoader != nil && b.configPath != "" {
		if err := b.configLoader.Load(b.configPath); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	for _, fn := range b.beforeInit {
		if err := fn(ctx); err != nil {
			return fmt.Errorf("before init failed: %w", err)
		}
	}

	if b.autoInit {
		if err := b.initInternalComponents(ctx); err != nil {
			return fmt.Errorf("failed to initialize internal components: %w", err)
		}
	}

	for _, component := range b.components {
		if err := component.Init(ctx); err != nil {
			return fmt.Errorf("failed to initialize component %s: %w", component.Name(), err)
		}
	}

	for _, fn := range b.afterInit {
		if err := fn(ctx); err != nil {
			return fmt.Errorf("after init failed: %w", err)
		}
	}

	return nil
}

func (b *Bootstrap) initInternalComponents(ctx context.Context) error {
	if b.configLoader == nil {
		configPath := b.configPath
		if configPath == "" {
			configPath = "config.yaml"
		}
		if err := config.InitGlobalConfigWithPath(configPath); err != nil {
			// If config file doesn't exist, use defaults (CLI mode)
			_ = config.InitGlobalConfigWithDefault()
		}
	}

	// Try DB init. If config has no DB config or DB unavailable, skip silently.
	// This allows CLI tools and non-DB services to use Bootstrap without DB.
	if err := db.InitGlobalDBWithDefault(); err != nil {
		return nil
	}

	// Init cache. Redis 可用时用 Redis，不可用时 fallback 到本地内存缓存。
	// 基于 tools/cache LRU 实现，确保离开 Redis 系统仍然可用。
	_ = cache.InitGlobalCacheWithDefault()

	// Auto-init JWT from config (optional).
	// jwt:` section in config.yaml is parsed into Config.JWT directly.
	if cfg := config.GetConfig(); cfg != nil && cfg.JWT.Secret != "" && cfg.JWT.ExpireTime != "" {
		expire, err := time.ParseDuration(cfg.JWT.ExpireTime)
		if err == nil {
			auth.InitJWT(&auth.JWTConfig{
				Secret:     cfg.JWT.Secret,
				ExpireTime: expire,
				Issuer:     cfg.JWT.Issuer,
			})
		}
	}

	return nil
}

func (b *Bootstrap) Run(ctx context.Context) error {
	for _, fn := range b.beforeRun {
		if err := fn(ctx); err != nil {
			return fmt.Errorf("before run failed: %w", err)
		}
	}

	for _, fn := range b.afterRun {
		if err := fn(ctx); err != nil {
			return fmt.Errorf("after run failed: %w", err)
		}
	}

	return nil
}

func (b *Bootstrap) Close() error {
	for i := len(b.components) - 1; i >= 0; i-- {
		component := b.components[i]
		if err := component.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close component %s: %v\n", component.Name(), err)
		}
	}
	db.Close()
	return nil
}
