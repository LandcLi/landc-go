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

## 快速开始

> 目标：10 分钟从零跑起第一个 HTTP 服务。本教程逐步覆盖：路由 → 参数绑定 → 统一响应 → 配置 → 数据库/缓存 → JWT 认证 → 工程化。
> 每段代码都可直接复制运行，并基于已验证的真实 API。

### 环境要求

- **Go 1.26+**（`frame` 模块要求；单独使用 `log` / `tools` / `api` 时版本要求更低）
- 可选的本地依赖：MySQL / Redis（教程提供了"无外部依赖也能跑通"的路径）

### 第 1 步：初始化项目

推荐用 `landc` CLI 一键生成带分层结构的标准项目骨架（自动 `git init` 与 `go mod tidy`）：

```bash
# 安装 landc CLI
go install github.com/LandcLi/landc-go/frame/cmd@latest

# 初始化项目
landc init myapp
cd myapp
```

生成的骨架：

```
myapp/
├── main.go                    # 入口：cmd.Main.Run(ctx)
├── config.yaml                # 服务配置（server/database/redis/jwt/log）
├── api/                       # 接口定义（Controller 接口 + v1 请求/响应）
│   └── hello/
├── service/                   # Service 接口
├── dao/                       # DAO 接口
├── model/                     # 数据模型
├── internal/
│   ├── cmd/cmd.go             # 主命令：注册控制器 + RunWithContext
│   ├── controller/hello/      # Controller 实现（init() 注册到 DI）
│   ├── service_impl/hello/    # Service 实现
│   ├── dao_impl/hello/        # DAO 实现
│   └── impl.go                # 触发所有子包 init() 完成 DI 注册
└── sqls/init.sql
```

> 想从零手动搭也可以：`go mod init myapp && go get github.com/LandcLi/landc-go/frame`，再参考下方各节逐步添加。
> 只需要某个独立模块时：`go get github.com/LandcLi/landc-go/log|tools|api`。

### 第 2 步：运行项目并验证

init 骨架自带一个 hello 示例（完整走通 Controller → Service → DAO 三层），直接运行：

```bash
go run main.go

curl http://localhost:8080/health        # 健康检查（config.yaml 已开启默认路由）
curl -X POST http://localhost:8080/api/hello/say \
  -H "Content-Type: application/json" \
  -d '{"name":"landc"}'
# 返回：{"code":10000,"message":"success","data":{"message":"Hello, landc!"}}
```

**路由从哪来？** 请求结构体通过内嵌 `meta.Meta` 声明路由（`api/hello/v1/say_hello.go`）：

```go
type SayHelloRequest struct {
    meta.Meta `path:"/api/hello/say" method:"POST"` // 路由声明
    Name      string `json:"name" binding:"required"`
}
```

Controller 方法签名带 `ctx context.Context`，框架自动绑定参数、调用、返回统一 JSON：

```go
func (c *helloController) SayHello(ctx context.Context, req *v1.SayHelloRequest) (*v1.SayHelloResponse, error) {
    return service.GetHelloService().SayHello(ctx, req)
}
```

各层通过 DI 容器注册/获取（`internal/impl.go` 引入子包触发 `init()` 注册链）。**新增一个接口**只需：定义 v1 请求/响应 → Controller 接口加方法 → Service/DAO 接口加方法 → 在 `internal/` 下补实现。

**理解最小原理**：一个接口可以只用几行代码，下面是等价的极简版（单文件即可跑）：

```go
package main

import (
    "github.com/LandcLi/landc-go/frame/pkg/meta"
    "github.com/LandcLi/landc-go/frame/pkg/middleware"
    "github.com/LandcLi/landc-go/frame/pkg/web"
)

// 请求结构体：内嵌 meta.Meta 声明方法路由
type HelloReq struct {
    meta.Meta `path:"/hello" method:"GET" description:"打招呼"`
    Name      string `source:"query" name:"name" d:"World"` // query 参数，默认值 World
}

type HelloRes struct {
    Message string `json:"message"`
}

// Controller：内嵌 meta.Meta 声明路由前缀
type HelloController struct {
    meta.Meta `path:"/api"`
}

// 方法即接口：框架自动绑定参数、调用、返回 JSON
func (c *HelloController) SayHello(req *HelloReq) (*HelloRes, error) {
    return &HelloRes{Message: "Hello, " + req.Name + "!"}, nil
}

func main() {
    server := web.NewServer(nil)
    server.Engine().Use(
        middleware.Trace(),    // 链路追踪（X-Trace-ID）
        middleware.Logger(),   // 结构化访问日志
        middleware.Recovery(), // panic 恢复
    )
    if err := server.RegisterHandler(&HelloController{}); err != nil {
        panic(err)
    }
    if err := server.Run(); err != nil {
        panic(err)
    }
}
```

框架已经帮你完成了：路由注册（`GET /api/hello`）、参数绑定与默认值、统一 JSON 响应、优雅停机。

### 第 3 步：路由与参数绑定

**Meta Tag**（内嵌 `meta.Meta` 的结构体 Tag）：

| Tag | 位置 | 说明 |
|-----|------|------|
| `path` | Controller / 请求结构体 | Controller 为路由前缀，请求结构体为方法路径 |
| `method` | 请求结构体 | HTTP 方法（GET/POST/PUT/DELETE/PATCH/...） |
| `description` | 请求结构体 | 接口说明（用于 OpenAPI 文档） |

最终路由 = 前缀 + 路径，如 `GET /api/hello`。

**参数来源**（字段上的 `source` Tag，默认 `body`）：

| Tag | 来源 | 示例 |
|-----|------|------|
| `source:"query"` | URL Query | `/users?page=1` |
| `source:"path"` | URL Path 参数 | `/users/:id` |
| `source:"header"` | HTTP Header | `X-Client-IP` |
| `source:"form"` | 表单 | `name=xxx` |
| `source:"body"` 或不写 | JSON Body | `{"name":"..."}` |
| `d:"默认值"` | 零值时自动填充 | `d:"1"` |

一个包含多种来源的完整示例：

```go
type LoginReq struct {
    meta.Meta `path:"/login" method:"POST" description:"登录"`
    Username  string `source:"body" name:"username" binding:"required"`
    Password  string `source:"body" name:"password" binding:"required"`
    ClientIP  string `source:"header" name:"X-Client-IP"`
    Redirect  string `source:"query" name:"redirect" d:"/home"`
}

type GetUserReq struct {
    meta.Meta `path:"/users/:id" method:"GET" description:"获取用户"`
    ID        uint `source:"path" name:"id" binding:"required"`
}
```

校验：支持 `binding:"required"`（go-playground/validator）以及 landc-go 的 `validate:"email,phone,min:6,length:8"` 等内置校验器（逗号/空格分隔）。

### 第 4 步：统一响应与错误码

框架对 Controller 方法返回值做了统一约定：

| 方法返回 | 框架行为 |
|---------|---------|
| `(data, nil)` | HTTP 200：`{"code":10000,"message":"success","data":...}` |
| `(nil, err)` | HTTP 500：`{"code":50000,"message":"internal server error"}`（**不泄露内部错误细节**，完整错误打印到服务端日志） |

```go
// 方式一：直接返回 error —— 业务失败统一映射为 500，内部细节不外泄
func (c *UserController) Delete(req *DeleteUserReq) (*DeleteUserRes, error) {
    if err := userService.Delete(req.ID); err != nil {
        return nil, err
    }
    return &DeleteUserRes{OK: true}, nil
}

// 方式二：需要精确的错误码/HTTP 状态 —— 方法首参注入 *gin.Context，
// 使用 response 包写出，并 Abort 阻止框架二次写响应
func (c *UserController) Create(g *gin.Context, req *CreateUserReq) (*CreateUserRes, error) {
    if req.Username == "" {
        response.BadRequest(g, "username is required") // HTTP 200 + code 40000
        g.Abort()
        return nil, nil
    }
    if err := userService.Create(req); err != nil {
        response.InternalServerError(g, "create failed") // code 50000
        g.Abort()
        return nil, nil
    }
    return &CreateUserRes{ID: 1}, nil
}
```

`response` 包提供 `Success / BadRequest / Unauthorized / Forbidden / NotFound / InternalServerError / Error(code, msg)` 等，错误码体系与 `github.com/LandcLi/landc-go/api/core` 完全一致（成功 10000，客户端错误 4xxxx，服务端错误 5xxxx，自定义错误 60000-99999）。

### 第 5 步：配置加载

`frame` 内置统一配置结构，支持 YAML / JSON 文件 + 环境变量覆盖 + 热更新。

新建 `config.yaml`：

```yaml
server:
  addr: 0.0.0.0
  port: 8080
  use_default_routes: true     # 自动注册 /health、/ready 健康检查
  health_check:
    enabled: true
    liveness_path: /health
    readiness_path: /ready
    database_check: true
    redis_check: false
  request_timeout: 10          # 秒，>0 时启用请求超时中间件
database:
  driver: mysql
  dsn: "root:password@tcp(localhost:3306)/myapp?charset=utf8mb4&parseTime=True&loc=Local"
redis:
  addr: localhost:6379
jwt:
  secret: ""                   # 生产从环境变量注入，见下文
  expire_time: 2h
  issuer: myapp
log:
  level: info
  format: json
  output: stdout
```

推荐用 `bootstrap` 一键初始化（自动完成：加载配置 → 初始化 DB → 初始化缓存 → 初始化 JWT → 启动配置热更新）：

```go
package main

import (
    "context"

    "github.com/LandcLi/landc-go/frame/pkg/bootstrap"
    "github.com/LandcLi/landc-go/frame/pkg/middleware"
    "github.com/LandcLi/landc-go/frame/pkg/web"
)

func main() {
    boot := bootstrap.New()
    boot.SetConfigPath("config.yaml")
    if err := boot.Init(context.Background()); err != nil {
        panic(err)
    }

    server := web.NewServer(nil) // 自动读取全局配置：端口 / 超时 / 健康检查
    server.Engine().Use(middleware.Trace(), middleware.Logger(), middleware.Recovery())
    server.RegisterHandler(&HelloController{})
    server.Run()
}
```

**环境变量覆盖**（优先级高于配置文件），命名规则 `LANDC_{SECTION}_{KEY}`：

```bash
export LANDC_SERVER_PORT=9090
export LANDC_DATABASE_DSN='root:pwd@tcp(localhost:3306)/myapp?charset=utf8mb4'
export LANDC_REDIS_ADDR=localhost:6380
export LANDC_LOG_LEVEL=debug
```

不用 `bootstrap` 时也可以手动加载：`config.InitGlobalConfigWithPathAndEnv("config.yaml")`。

### 第 6 步：接入数据库与缓存

初始化（使用全局配置中的 `database` / `redis` 段）：

```go
import (
    "github.com/LandcLi/landc-go/frame/pkg/cache"
    "github.com/LandcLi/landc-go/frame/pkg/db"
)

// 数据库：基于 GORM，支持 mysql / postgres / sqlite
if err := db.InitGlobalDBWithDefault(); err != nil { /* 处理错误 */ }

// 缓存：Redis 可用则用 Redis，不可用自动回退本地内存缓存（弱依赖模式）
cache.InitGlobalCacheWithDefault()
```

定义模型并自动建表：

```go
type User struct {
    ID        uint      `gorm:"primarykey" json:"id"`
    Name      string    `gorm:"type:varchar(100);not null" json:"name"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

func (User) TableName() string { return "users" }

db.AutoMigrate(&User{})
```

**在 DAO / Service 中访问数据库**（推荐 `db.GetTx(ctx)`：有事务时自动用事务连接，否则用普通连接）：

```go
func (d *userDaoImpl) Create(ctx context.Context, u *User) error {
    return db.GetTx(ctx).Create(u).Error
}

// 事务：多个写操作共享同一连接
err := db.Transaction(ctx, func(ctx context.Context) error {
    if err := db.GetTx(ctx).Create(u).Error; err != nil {
        return err
    }
    return db.GetTx(ctx).Model(u).Update("name", "new").Error
})
```

**缓存读写**：

```go
c := cache.GetCache()
c.Set(ctx, "user:"+key, u, 10*time.Minute)          // 自动 JSON 序列化
var cached User
c.GetObject(ctx, "user:"+key, &cached)
c.Incr(ctx, "login:daily:"+ip, 24*time.Hour)        // 原子自增（限流/计数用）
```

> 注意：Controller 方法签名里带上 `ctx context.Context`，框架会传入请求上下文（含资源作用域），Service/DAO 全程透传即可。

### 第 7 步：JWT 认证与中间件

**初始化与签发**（HS256 密钥必须 ≥ 32 字符）：

```go
import (
    "github.com/LandcLi/landc-go/frame/pkg/auth"
    "github.com/LandcLi/landc-go/frame/pkg/middleware"
    "github.com/LandcLi/landc-go/frame/pkg/web"
)

// 在启动时初始化（或由 bootstrap 从 config.yaml 的 jwt 段自动完成）
auth.InitJWT(&auth.JWTConfig{
    Secret:     os.Getenv("APP_JWT_SECRET"), // 生产从环境变量注入，≥32 字符
    ExpireTime: 2 * time.Hour,
    Issuer:     "myapp",
})

// 登录成功后签发：
token, err := auth.GenerateToken(10001, "landc", "admin") // userID, username, role

// 解析校验：
claims, err := auth.ParseToken(token) // *auth.Claims：UserID/Username/Role/ClientID/Scope...
```

**保护接口**（三种粒度）：

```go
// 1) 全局：所有接口都要求登录
server.Engine().Use(middleware.Auth())

// 2) 方法级：只保护特定接口（推荐，灵活）
server.RegisterHandler(&UserController{},
    web.WithMethodMiddleware("GetProfile", middleware.Auth()),
    web.WithMethodMiddleware("DeleteUser", middleware.Auth(), middleware.RoleRequired("admin")),
)

// 3) 控制器级：整个 Controller 都要求登录
server.RegisterHandler(&UserController{},
    web.WithControllerMiddleware(middleware.Auth()),
)
```

Controller 内通过 `*gin.Context` 读取已认证的用户信息：

```go
func (c *UserController) GetProfile(g *gin.Context, req *GetProfileReq) (*UserRes, error) {
    uid := g.GetUint("user_id") // 中间件写入
    role := g.GetString("role")
    // ...
}
```

### 第 8 步：分层工程与代码生成

推荐目录结构（API → Controller → Service → DAO，各层通过接口解耦）：

```
myapp/
├── cmd/server/main.go      # 入口：bootstrap + 注册控制器
├── internal/
│   ├── controller/         # 控制器：Meta Tag 路由 + 参数绑定
│   ├── service/            # 业务逻辑（接口 + 实现）
│   ├── dao/                # 数据访问（db.GetTx）
│   ├── model/              # 数据模型 / 输入输出
│   └── api/xxx/v1/         # 请求 / 响应结构体
└── config.yaml
```

在 init 生成的骨架基础上，用 `landc` CLI 生成新实体（如 User）的四层代码：

```bash
landc gen all User            # 一次性生成 api/service/dao/model 四层
landc gen all User --check    # 生成后自动执行 go build ./... 校验可编译
landc gen api User            # 只生成 API 层
landc gen service User        # 只生成 Service 层
landc gen dao User            # 只生成 DAO 层
landc gen model User          # 只生成 model 层
landc gen lib user            # 生成库模式入口 serverlib/RegisterToRouter 骨架
landc gen proxy --type UserController  # 生成远程代理 SDK（NewXXXProxy）
landc migrate-dbctx [--dry-run] <dir>  # 存量代码迁移 ctx（--dry-run 预览不写盘）
```

生成的代码会自动 `gofmt` 格式化，无需手动处理。

### 本地开发

```bash
# 在 landc-go 仓库内使用 go.work 启用多模块联调
go work use ./api ./frame ./log ./tools ./workflow ./saas

# 常用开发命令
make fmt           # 格式化所有代码
make vet           # 代码静态检查
make test          # 运行所有测试
make lint          # 运行 golangci-lint
make build         # 编译所有模块
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
landc version                          # 版本信息
landc doctor [--check-network]         # 环境诊断（go 版本 / GOPROXY / 依赖可解析性）
landc init myapp                       # 初始化分层项目骨架
landc gen all User                     # 生成 api/service/dao/model 四层（--check 生成后 go build 校验）
landc gen api|service|dao|model User   # 单独生成某一层
landc gen lib user                     # 生成库模式入口 serverlib/RegisterToRouter 骨架
landc gen proxy --type UserController  # 生成远程代理 SDK（含 NewXXXProxy 显式构造）
landc migrate-dbctx [--dry-run] <dir>  # DAO/service 迁移 ctx（--dry-run 预览不写盘）
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
│   ├── cmd/                 landc CLI（main.go 入口）
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
│       └── cmd/             CLI 命令行框架
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
