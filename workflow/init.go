package workflow

import (
	"fmt"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/cache"
	"github.com/LandcLi/landc-go/frame/pkg/db"
	"github.com/LandcLi/landc-go/log/facade"
	"github.com/LandcLi/landc-go/workflow/pkg/engine"
	"github.com/LandcLi/landc-go/workflow/pkg/executor"
	"github.com/LandcLi/landc-go/workflow/pkg/idempotent"
	"github.com/LandcLi/landc-go/workflow/pkg/observer"
	storepkg "github.com/LandcLi/landc-go/workflow/pkg/store"
	"gorm.io/gorm"
)

// Init 从框架全局配置初始化工作流框架。
// 前置条件：
//   - frameconfig.InitGlobalConfigWithPath() 已调用
//   - db 已初始化（由框架自动完成）
//   - cache 已初始化（可选，有 Redis 时自动使用 Redis 做幂等和锁）
func Init() error {
	cfg := readConfig()

	gormDB := db.GetDB()
	if gormDB == nil {
		panic("workflow: DB not initialized, call db.InitGlobalDB first")
	}

	dbStore := storepkg.NewDBStore(gormDB)
	if err := dbStore.AutoMigrate(); err != nil {
		return err
	}

	// 通过 DI 注册 Store
	storepkg.RegisterStore(dbStore)

	// 注册执行器
	execReg := executor.NewRegistry()
	executor.RegisterDefault(execReg)

	// 创建观察者 — 复用 log/facade
	wfLog := facade.GetLoggerWithName("workflow")
	obsManager := observer.NewObserverManager()
	obsManager.Register(observer.NewTraceObserver())
	obsManager.Register(observer.NewLogObserver(wfLog))

	// 选择幂等检查器：Redis 可用时用 Redis 版，否则用内存版
	var idempCheck idempotent.IdempotencyChecker
	ttl := time.Duration(cfg.Engine.IdempotencyTTL) * time.Second
	if redisClient := cache.GetRedis(); redisClient != nil {
		redisStore := storepkg.NewRedisStore("wf")
		idempCheck = idempotent.NewStoreIdempotencyChecker(redisStore, ttl)
	} else {
		idempCheck = idempotent.NewMemoryIdempotencyChecker(ttl)
	}

	// 创建引擎
	eng := engine.NewEngine(
		dbStore, execReg, obsManager, idempCheck,
		engine.EngineConfig{
			MaxParallelTasks: cfg.Engine.MaxParallelTasks,
			DefaultTimeout:   time.Duration(cfg.Engine.DefaultTimeout) * time.Second,
			IdempotencyTTL:   ttl,
		},
	)

	// 通过 DI 注册引擎
	engine.RegisterEngine(eng)

	return nil
}

// MustInit 如 Init 但失败时 panic（适合 main() 中调用）
func MustInit() {
	if err := Init(); err != nil {
		panic("workflow init failed: " + err.Error())
	}
}

// InitWithComponents 使用手动传入的组件初始化（不依赖框架全局配置）。
// 适用于测试或不想使用框架 DB/Cache 全局单例的场景。
func InitWithComponents(gormDB *gorm.DB) error {
	if gormDB == nil {
		return fmt.Errorf("workflow: gormDB must not be nil")
	}

	dbStore := storepkg.NewDBStore(gormDB)
	if err := dbStore.AutoMigrate(); err != nil {
		return err
	}
	storepkg.RegisterStore(dbStore)

	execReg := executor.NewRegistry()
	executor.RegisterDefault(execReg)

	wfLog := facade.GetLoggerWithName("workflow")
	obsManager := observer.NewObserverManager()
	obsManager.Register(observer.NewLogObserver(wfLog))

	idempCheck := idempotent.NewMemoryIdempotencyChecker(24 * time.Hour)

	eng := engine.NewEngine(dbStore, execReg, obsManager, idempCheck, engine.DefaultEngineConfig())
	engine.RegisterEngine(eng)
	return nil
}
