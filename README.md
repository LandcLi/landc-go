# landc-go

**landc-go** 是一套 Go 语言应用开发框架，采用多 Module 单仓库（Monorepo）架构，各模块可独立引用、独立发版。

[![CI](https://github.com/LandcLi/landc-go/actions/workflows/ci.yml/badge.svg)](https://github.com/LandcLi/landc-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/LandcLi/landc-go.svg)](https://pkg.go.dev/github.com/LandcLi/landc-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/LandcLi/landc-go)](https://goreportcard.com/report/github.com/LandcLi/landc-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## 模块概览

| 模块 | Import 路径 | 说明 | 可独立使用 |
|------|------------|------|-----------|
| **log** | `github.com/LandcLi/landc-go/log` | 日志门面，统一接口支持 Zap / Logrus / Console | ✅ |
| **tools** | `github.com/LandcLi/landc-go/tools` | 通用工具集（缓存 / 字符串 / 数字 / UUID / 地理 / 邮件 / Tag 验证 / **AES-GCM 加解密 / 缓存限流 / 验证码**） | ✅ |
| **api** | `github.com/LandcLi/landc-go/api` | API 规范（统一响应结构 / 错误码体系 / 框架中间件） | ✅ |
| **frame** | `github.com/LandcLi/landc-go/frame` | Web 全栈框架（Web / DI / ORM / Redis / JWT / gRPC / WebSocket / **命名资源作用域 / 限流 / 验证码** / ...） | 依赖 api+log+tools |
| **workflow** | `github.com/LandcLi/landc-go/workflow` | 分布式工作流引擎（DAG 编排 / 节点执行 / 状态管理） | 依赖 frame |
| **saas** | `github.com/LandcLi/landc-go/saas` | 多租户 SaaS 插件（租户隔离 / 数据分片） | 依赖 frame |

## 快速上手

```bash
# 只需要日志能力
go get github.com/LandcLi/landc-go/log

# 只需要工具函数
go get github.com/LandcLi/landc-go/tools

# 只需要 API 规范
go get github.com/LandcLi/landc-go/api

# 使用完整 Web 全栈框架
go get github.com/LandcLi/landc-go/frame
```

### 本地开发

```bash
# 使用 go.work 启用多模块开发
go work use ./api ./frame ./log ./tools ./workflow ./saas

# 常用开发命令
make fmt           # 格式化所有代码
make vet           # 代码静态检查
make test          # 运行所有测试
make lint          # 运行 golangci-lint
make build         # 编译所有模块
```

### 使用示例

```go
package main

import (
    "github.com/LandcLi/landc-go/log/facade"
    "github.com/LandcLi/landc-go/frame/pkg/auth"
    "github.com/LandcLi/landc-go/frame/pkg/middleware"
    "github.com/LandcLi/landc-go/frame/pkg/web"
)

func main() {
    facade.Info("starting server...")

    server := web.NewServer(nil)
    server.Engine().Use(
        middleware.Trace(),
        middleware.Logger(),
        middleware.Recovery(),
    )
    server.RegisterHandler(&MyController{})
    server.Run()
}
```

## 核心能力

### 路由注册选项（v0.6.0）

运行时覆盖路由与挂载中间件，优先于编译期 Meta Tag；不传选项行为与旧版完全一致：

```go
server.RegisterHandler(&MyController{},
    web.WithPrefix("/api/v2"),
    web.WithMethodPath("Login", "/sso/login"),
    web.WithMethodMiddleware("Login", rateLimiter),      // 多中间件按序执行
    web.WithControllerMiddleware(auditor),
)
```

### 路由查询（v0.6.0）

```go
routes := server.Routes() // []web.RouteInfo{Method, Path, Description, HandlerName}，最终生效路由
```

### OpenAPI 自动生成（v0.6.0）

```go
gen := openapi.NewGenerator(openapi.Info{Title: "My Service", Version: "1.0"})
gen.AddBearerAuth()
gen.RegisterServer(server) // 从已注册控制器自动收集（路径为最终生效路由）
gen.WriteFile("openapi.json") // CI 落盘
```

### 命名资源作用域（v0.6.0，库模式嵌入）

嵌入服务使用宿主为之准备的**独立** DB / 缓存 / 配置，未指定字段回退全局（不静默回退，防数据写错库）：

```go
// 宿主注册命名资源
db.InitNamedDB("lum", &dbConfig)
cache.InitNamedCacheWithLocal("lum", 1000)

// 库模式注册嵌入服务（自动注入作用域，controller 经 ctx 解析）
server.RegisterLibrary(embeddedCtrl,
    web.WithScope(resource.Scope{Name: "lum", DB: "lum", Cache: "lum"}),
)

// controller 方法签名带 context.Context 时，经 ctx 解析命名资源：
func (c *Ctrl) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
    gdb := db.GetDBFrom(ctx) // 命名连接；无作用域回退全局
}
```

### 缓存限流与验证码（v0.7.0）

```go
// 缓存限流（间隔 / 计数，Cache.Incr 原子自增防并发竞态）
if !ratelimit.AllowInterval(ctx, "sms:send:"+phone, 60*time.Second) { ... }
if !ratelimit.AllowCount(ctx, "login:daily:"+ip, 10, 24*time.Hour) { ... }

// 验证码（生成含限流与 TTL 存储；校验一次性防重放）
code, _ := verifycode.Generate(ctx, "sms:13800000000")
if !verifycode.Verify(ctx, "sms:13800000000", input) { ... }
```

### AES-GCM 加解密（v0.7.0）

```go
cipher, _ := security.NewCipherFromString(os.Getenv("APP_CRYPTO_KEY")) // 16/24/32 字节
enc, _ := cipher.Encrypt([]byte("sensitive"))
plain, _ := cipher.Decrypt(enc) // 兼容无 enc: 前缀的历史明文
```

### CLI 工具

```bash
landc gen proxy --type UserController   # 生成远程代理 SDK（含 NewXXXProxy 显式构造）
landc gen lib user                      # 生成库模式入口 serverlib/RegisterToRouter 骨架
landc migrate-dbctx <dir>               # DAO/service 迁移 ctx（GetDB()→GetDBFrom(ctx) 等）
```

## 版本管理

每个模块独立发版，使用路径前缀 Git Tag：

```bash
git tag log/v0.1.0 && git push origin log/v0.1.0
git tag tools/v0.1.0 && git push origin tools/v0.1.0
git tag api/v0.1.0 && git push origin api/v0.1.0
git tag frame/v0.1.0 && git push origin frame/v0.1.0
git tag workflow/v0.1.0 && git push origin workflow/v0.1.0
git tag saas/v0.1.0 && git push origin saas/v0.1.0
```

## 架构设计

```
依赖方向（单向，无循环）:

    saas  →  frame  →  api  →  tools
                   │
                   └──→  log
```

## 目录结构

```
landc-go/
├── .github/workflows/      # CI/CD 流水线
│   └── ci.yml
├── .golangci.yml            # Lint 配置
├── .gitignore               # Git 忽略规则
├── .env.example             # 环境变量模板
├── Makefile                 # 构建/测试/lint 统一入口
├── go.work                  # Go 工作区（本地多模块开发）
├── Dockerfile               # 容器化多阶段构建
├── LICENSE                  # MIT 许可
├── README.md
├── CONTRIBUTING.md          # 贡献指南
├── CODE_OF_CONDUCT.md       # 行为准则
├── SECURITY.md              # 安全策略
├── log/                     # 日志门面
│   ├── go.mod               module github.com/LandcLi/landc-go/log
│   ├── facade/              公开 API（Info/Debug/Warn/Error/...）
│   ├── internal/logger/     核心实现
│   ├── providers/           后端适配（zap / logrus）
│   ├── adapter/             框架集成（gin / goframe）
│   └── tests/
├── tools/                   # 工具集
│   ├── go.mod               module github.com/LandcLi/landc-go/tools
│   ├── cache/               本地缓存（LRU + 过期清理）
│   ├── str/                 字符串处理
│   ├── num/                 数字处理
│   ├── generate/            UUID / 随机数 / 雪花 ID
│   ├── converter/           类型转换
│   ├── email/               邮箱格式验证
│   ├── geo/                 IP 地理定位
│   ├── mail/                SMTP 邮件收发
│   ├── security/            AES-GCM 加解密（enc: 前缀 + 明文兼容降级）
│   ├── ratelimit/           缓存限流（间隔 / 计数，Cache 接口原子自增）
│   ├── verifycode/          验证码（生成 / 限流 / 一次性校验）
│   └── tag/                 结构体 Tag 验证器
├── api/                     # API 规范
│   ├── go.mod               module github.com/LandcLi/landc-go/api
│   ├── core/                Response / ErrorCode / Error
│   ├── middleware/          Gin / GoFrame 中间件
│   └── trace/               链路追踪
├── frame/                   # Web 全栈框架
│   ├── go.mod               module github.com/LandcLi/landc-go/frame
│   └── pkg/
│       ├── web/             Web 引擎（Meta Tag 路由 / 参数绑定 / 文件上传）
│       ├── di/              依赖注入容器
│       ├── config/          配置管理（YAML / 环境变量 / 热更新）
│       ├── db/              数据库（GORM + 迁移工具）
│       ├── cache/           Redis 缓存（含原子 Incr / AsToolsCache 适配）
│       ├── resource/        命名资源作用域（库模式嵌入独立配置/DB/缓存）
│       ├── ratelimit/       缓存限流包装（从 ctx 解析命名缓存）
│       ├── verifycode/      验证码包装（从 ctx 解析命名缓存）
│       ├── auth/            JWT 认证
│       ├── middleware/      中间件（Logger / Recovery / CORS / Auth / 限流 / 熔断）
│       ├── response/        统一响应格式
│       ├── i18n/            国际化
│       ├── session/         Session（Redis / Memory）
│       ├── websocket/       WebSocket（Hub / 房间 / 广播）
│       ├── cron/            定时任务
│       ├── openapi/         OpenAPI 3.0 文档自动生成
│       ├── registry/        服务注册 / 发现（etcd）
│       ├── grpc/            gRPC 服务端 / 客户端
│       ├── gen/             代码生成（landc gen api/service/dao/lib）
│       ├── codemod/         代码迁移（landc migrate-dbctx）
│       ├── trace/           链路追踪
│       ├── meta/            元数据
│       ├── bootstrap/       应用生命周期
│       └── cmd/             CLI 命令行（gen / gen proxy / migrate-dbctx）
├── workflow/                # 分布式工作流引擎
│   └── go.mod               module github.com/LandcLi/landc-go/workflow
└── saas/                    # 多租户 SaaS 插件
    └── go.mod               module github.com/LandcLi/landc-go/saas
```

## 设计理念

- **仿 GoFrame 风格**：Controller 通过 Meta Tag 声明路由，请求参数结构体自动绑定
- **分层架构**：API → Controller → Service → DAO，各层通过接口解耦
- **模块化**：各基础模块可独立使用，不强制绑定完整框架
- **约定优于配置**：合理的默认值，最小化样板代码

## 贡献

欢迎贡献代码！请参阅 [CONTRIBUTING.md](CONTRIBUTING.md) 了解如何开始。

## License

MIT
