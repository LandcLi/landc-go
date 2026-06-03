// Package demo 展示 landc-go/frame 框架的完整用法
// 这是一个用户管理 API 示例，演示分层架构（API -> Service -> DAO）
package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/auth"
	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/LandcLi/landc-go/frame/pkg/middleware"
	"github.com/LandcLi/landc-go/frame/pkg/web"
	"github.com/gin-gonic/gin"
)

// Run 启动示例应用
func Run() {
	// 1. 初始化配置
	config.InitGlobalConfigWithConfig(&config.Config{
		Server: config.ServerConfig{
			Addr: "0.0.0.0",
			Port: 8080,
		},
		Log: config.LogConfig{
			Level:  "debug",
			Format: "text",
			Output: "stdout",
		},
	})

	// 2. 初始化 JWT
	auth.InitJWT(&auth.JWTConfig{
		Secret:     "demo-secret-key-change-in-production",
		ExpireTime: 2 * time.Hour,
		Issuer:     "landc-demo",
	})

	// 3. 注册依赖（Service -> DAO）
	RegisterDependencies()

	// 4. 创建 Web 服务器
	server := web.NewServer(&web.ServerConfig{
		Addr: ":8080",
	})

	// 5. 配置中间件
	engine := server.Engine()
	engine.Use(middleware.Trace())
	engine.Use(middleware.Logger())
	engine.Use(middleware.Recovery())
	engine.Use(middleware.CORS())

	// 6. 注册路由
	RegisterRoutes(engine)

	// 7. 启动服务
	fmt.Println("Demo server starting on :8080")
	if err := server.Run(); err != nil {
		panic(err)
	}
}

// RegisterDependencies 注册所有依赖实现
func RegisterDependencies() {
	// 注册 DAO 实现（使用内存存储作为 demo）
	userDAO := NewMemoryUserDAO()
	RegisterUserDAO(userDAO)

	// 注册 Service 实现
	userService := NewUserServiceImpl()
	RegisterUserService(userService)

	// 注册 Controller 实现
	userController := NewUserController()
	RegisterUserController(userController)
}

// RegisterRoutes 注册路由
func RegisterRoutes(r *gin.Engine) {
	// 健康检查（无需认证）
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 登录（无需认证）
	r.POST("/api/v1/login", Login)

	// 需要认证的路由
	authorized := r.Group("/api/v1")
	authorized.Use(middleware.Auth())
	{
		authorized.POST("/user/create", CreateUser)
		authorized.GET("/user/get", GetUser)
		authorized.PUT("/user/update", UpdateUser)
		authorized.DELETE("/user/delete", DeleteUser)
		authorized.GET("/user/list", ListUsers)
	}
}

// ============ 启动入口 ============

// Init 初始化（供外部包通过 import 触发）
func Init(ctx context.Context) error {
	RegisterDependencies()
	return nil
}
