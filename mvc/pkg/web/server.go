package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LandcLi/landc-go/mvc/pkg/bootstrap"
	"github.com/LandcLi/landc-go/log/facade"
	"github.com/gin-gonic/gin"
)

type (
	Server struct {
		engine    *gin.Engine
		config    *ServerConfig
		bootstrap *bootstrap.Bootstrap
	}

	ServerConfig struct {
		Addr             string
		ReadTimeout      time.Duration
		WriteTimeout     time.Duration
		ShutdownTimeout  time.Duration
		Handler          http.Handler
	}

	RouterGroup struct {
		group *gin.RouterGroup
	}
)

func NewServer(config *ServerConfig) *Server {
	if config == nil {
		config = &ServerConfig{}
	}
	if config.Addr == "" {
		config.Addr = ":8080"
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 60 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 60 * time.Second
	}
	if config.ShutdownTimeout == 0 {
		config.ShutdownTimeout = 10 * time.Second
	}

	return &Server{
		engine:    gin.Default(),
		config:    config,
		bootstrap: bootstrap.New(),
	}
}

func (s *Server) Engine() *gin.Engine {
	return s.engine
}

func (s *Server) Bootstrap() *bootstrap.Bootstrap {
	return s.bootstrap
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

// RunWithBootstrap 使用 Bootstrap 启动（包含初始化和优雅停机）
func (s *Server) RunWithBootstrap(addr ...string) error {
	ctx := context.Background()
	if err := s.bootstrap.Init(ctx); err != nil {
		return fmt.Errorf("bootstrap init failed: %w", err)
	}
	defer s.bootstrap.Close()

	return s.Run(addr...)
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
