package cmd

import (
	"context"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
	"github.com/LandcLi/landc-go/frame/pkg/db"
	"github.com/LandcLi/landc-go/frame/pkg/middleware"
	"github.com/LandcLi/landc-go/frame/pkg/web"

	"github.com/LandcLi/landc-go/examples/demo/api/auth"
	"github.com/LandcLi/landc-go/examples/demo/api/hello"
	"github.com/LandcLi/landc-go/examples/demo/model"
)

// Main 是 demo 主命令：bootstrap 生命周期（config/db/cache/jwt）由 cmd.NewCommand 自动完成。
var Main = cmd.NewCommand("server", "run landc-go demo server", func(ctx context.Context, parser *cmd.Parser) error {
	// 自动建表（SQLite 零配置）
	if err := db.AutoMigrate(&model.Hello{}); err != nil {
		return err
	}

	server := web.NewServer(nil)
	server.Engine().Use(
		middleware.Trace(),    // 链路追踪（X-Trace-ID）
		middleware.Logger(),   // 结构化访问日志
		middleware.Recovery(), // panic 恢复
	)

	// hello：普通路由
	if err := server.RegisterHandler(hello.GetHelloController()); err != nil {
		return err
	}

	// auth：Profile 方法挂载 JWT 认证中间件（方法级），演示 WithMethodMiddleware
	if err := server.RegisterHandler(auth.GetAuthController(),
		web.WithMethodMiddleware("Profile", middleware.Auth()),
	); err != nil {
		return err
	}

	return server.RunWithContext(ctx)
})
