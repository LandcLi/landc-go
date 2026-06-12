package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/LandcLi/landc-go/frame/pkg/health"
	"github.com/LandcLi/landc-go/frame/pkg/middleware"
	"github.com/LandcLi/landc-go/log/facade"
	"github.com/gin-gonic/gin"
)

type (
	Server struct {
		engine *gin.Engine
		config *ServerConfig
	}

	ServerConfig struct {
		Addr            string
		ReadTimeout     time.Duration
		WriteTimeout    time.Duration
		ShutdownTimeout time.Duration
		Handler         http.Handler
	}

	RouterGroup struct {
		group *gin.RouterGroup
	}
)

// New 使用默认配置创建 Server，自动读取全局配置（如有），否则使用内置默认值。
func New() *Server {
	return NewServer(nil)
}

// NewServer 使用指定配置创建 Server。
// 传 nil 时自动从全局配置读取（config.InitGlobalConfigWithPath），
// 未加载全局配置时使用内置默认值。
func NewServer(cfg *ServerConfig) *Server {
	if cfg == nil {
		// 自动从全局配置读取
		if globalCfg := config.GetConfig(); globalCfg != nil {
			cfg = &ServerConfig{
				Addr:         globalCfg.GetServerAddr(),
				ReadTimeout:  time.Duration(globalCfg.Server.ReadTimeout) * time.Second,
				WriteTimeout: time.Duration(globalCfg.Server.WriteTimeout) * time.Second,
			}
		}
	}
	if cfg == nil {
		cfg = &ServerConfig{}
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 60 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 60 * time.Second
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 10 * time.Second
	}

	s := &Server{
		engine: gin.Default(),
		config: cfg,
	}

	// 自动注册框架级中间件和默认路由（基于全局配置）
	s.registerBuiltin()

	return s
}

// registerBuiltin 自动注册框架级中间件（请求超时等）和默认路由（健康检查等）。
func (s *Server) registerBuiltin() {
	globalCfg := config.GetConfig()
	if globalCfg == nil {
		return
	}

	// 注册请求超时中间件
	if globalCfg.Server.RequestTimeout > 0 {
		facade.Info("request timeout middleware enabled",
			facade.Field{Key: "timeout_seconds", Value: globalCfg.Server.RequestTimeout})
		s.engine.Use(middleware.Timeout(
			time.Duration(globalCfg.Server.RequestTimeout) * time.Second,
		))
	}

	// 注册默认路由（健康检查）
	if globalCfg.Server.UseDefaultRoutes {
		s.registerDefaultRoutes(globalCfg.Server.HealthCheck)
	}
}

// registerDefaultRoutes 注册框架默认路由（健康检查端点）。
func (s *Server) registerDefaultRoutes(hc config.HealthCheckConfig) {
	if !hc.Enabled {
		return
	}

	// 注册内置检查器（DB、Redis）
	health.RegisterDefaultCheckers(hc.DatabaseCheck, hc.RedisCheck)

	// 存活性检查 — 轻量级，只返回 200
	livenessPath := hc.LivenessPath
	if livenessPath == "" {
		livenessPath = "/health"
	}
	s.engine.GET(livenessPath, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	facade.Info("liveness check registered",
		facade.Field{Key: "path", Value: livenessPath})

	// 启动性检查 — 可选，由 StartupPath 控制
	if hc.StartupPath != "" {
		s.engine.GET(hc.StartupPath, func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "serving"})
		})
		facade.Info("startup check registered",
			facade.Field{Key: "path", Value: hc.StartupPath})
	}

	// 就绪性检查 — 检查所有注册的 Checker
	readinessPath := hc.ReadinessPath
	if readinessPath == "" {
		readinessPath = "/ready"
	}
	s.engine.GET(readinessPath, s.handleReadiness())
	facade.Info("readiness check registered",
		facade.Field{Key: "path", Value: readinessPath})
}

// handleReadiness 处理就绪性检查请求，执行所有注册的 Checker。
func (s *Server) handleReadiness() gin.HandlerFunc {
	return func(c *gin.Context) {
		checkers := health.GlobalCheckers()
		results := make([]health.CheckResult, 0, len(checkers))
		overallStatus := "ok"

		for _, checker := range checkers {
			result := health.CheckResult{Name: checker.Name(), Status: "up"}
			if err := checker.Check(c.Request.Context()); err != nil {
				result.Status = "down"
				result.Error = err.Error()
				overallStatus = "degraded"
			}
			results = append(results, result)
		}

		statusCode := http.StatusOK
		if overallStatus == "degraded" {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, health.CheckResponse{
			Status: overallStatus,
			Checks: results,
		})
	}
}

func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// Run 启动服务（阻塞，支持优雅停机）
func (s *Server) Run(addr ...string) error {
	listenAddr := s.config.Addr
	if len(addr) > 0 {
		listenAddr = addr[0]
	}

	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      s.engine,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	// 启动 HTTP 服务（非阻塞）
	errCh := make(chan error, 1)
	go func() {
		facade.Info("server starting", facade.Field{Key: "addr", Value: listenAddr})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// 等待中断信号或启动错误
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("server start failed: %w", err)
	case sig := <-quit:
		facade.Info("shutdown signal received", facade.Field{Key: "signal", Value: sig.String()})
	}

	// 优雅停机
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	facade.Info("server shutting down...")
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	facade.Info("server stopped gracefully")
	return nil
}

// RunWithContext 使用上下文启动服务，ctx 取消时自动优雅停机
// 配合 cmd.Command.Run() 的信号处理使用：
//
//	cmd.NewCommand("main", "", func(ctx context.Context, parser *cmd.Parser) error {
//	    server := web.NewServer(nil)
//	    server.RegisterHandler(...)
//	    return server.RunWithContext(ctx)
//	})
func (s *Server) RunWithContext(ctx context.Context, addr ...string) error {
	listenAddr := s.config.Addr
	if len(addr) > 0 {
		listenAddr = addr[0]
	}

	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      s.engine,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		facade.Info("server starting", facade.Field{Key: "addr", Value: listenAddr})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server start failed: %w", err)
	case <-ctx.Done():
		facade.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	facade.Info("server shutting down...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	facade.Info("server stopped gracefully")
	return nil
}

// RunSimple 简单启动（不支持优雅停机，用于测试）
func (s *Server) RunSimple(addr ...string) error {
	listenAddr := s.config.Addr
	if len(addr) > 0 {
		listenAddr = addr[0]
	}
	return s.engine.Run(listenAddr)
}

func (s *Server) RegisterHandler(instance interface{}) error {
	return registerHandlers(s.engine, instance)
}

func (s *Server) Group(relativePath string, handlers ...gin.HandlerFunc) *RouterGroup {
	return &RouterGroup{
		group: s.engine.Group(relativePath, handlers...),
	}
}

func (rg *RouterGroup) RegisterHandler(instance interface{}) error {
	return registerHandlers(rg.group, instance)
}

func (rg *RouterGroup) Group(relativePath string, handlers ...gin.HandlerFunc) *RouterGroup {
	return &RouterGroup{
		group: rg.group.Group(relativePath, handlers...),
	}
}
