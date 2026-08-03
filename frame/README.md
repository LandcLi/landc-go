# landc-go/frame

仿 GoFrame 风格的 Go MVC 框架，基于 Gin 构建。

## 安装

```bash
go get github.com/LandcLi/landc-go/frame
```

## 核心特性

- **Meta Tag 路由注册** — Controller 方法通过结构体 Tag 声明路由，无需手动注册
- **参数自动绑定** — 支持 `source:"query/path/header/body"` + `d` 默认值 Tag
- **分层架构** — API → Controller → Service → DAO
- **依赖注入** — 泛型 DI 容器
- **统一响应** — 集成 landc-go/api 错误码体系
- **优雅停机** — 信号监听 + Shutdown

## 快速开始

```go
package main

import (
    "github.com/LandcLi/landc-go/frame/pkg/meta"
    "github.com/LandcLi/landc-go/frame/pkg/middleware"
    "github.com/LandcLi/landc-go/frame/pkg/web"
)

// 请求结构体：Meta Tag 定义路由
type HelloReq struct {
    meta.Meta `path:"/hello" method:"GET" description:"打招呼"`
    Name      string `source:"query" name:"name" d:"World"`
}

type HelloRes struct {
    Message string `json:"message"`
}

// Controller：Meta Tag 定义路由前缀
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
        middleware.Trace(),
        middleware.Logger(),
        middleware.Recovery(),
    )
    // 一行注册，自动生成路由 GET /api/hello
    server.RegisterHandler(&HelloController{})
    server.Run()
}
```

## 功能模块

| 包 | 功能 |
|----|------|
| `pkg/web` | Web 引擎（Meta 路由 / 参数绑定 / 文件上传） |
| `pkg/di` | 依赖注入（泛型容器 / 懒加载单例） |
| `pkg/config` | 配置管理（YAML / 环境变量覆盖 / 热更新） |
| `pkg/db` | 数据库（GORM 集成 / 事务 / 分页 / 版本化迁移） |
| `pkg/cache` | Redis 缓存 |
| `pkg/auth` | JWT 认证 |
| `pkg/middleware` | 中间件（Logger / Recovery / CORS / Trace / Auth / 限流 / 熔断） |
| `pkg/response` | 统一响应格式 |
| `pkg/i18n` | 国际化（JSON/YAML 语言文件 / Accept-Language） |
| `pkg/session` | Session（Redis / Memory） |
| `pkg/websocket` | WebSocket（Hub / 房间 / 广播 / 心跳） |
| `pkg/cron` | 定时任务 |
| `pkg/openapi` | OpenAPI 3.0 文档自动生成 + Swagger UI |
| `pkg/registry` | 服务注册 / 发现（etcd） |
| `pkg/grpc` | gRPC 服务端 / 客户端 / 连接池 |
| `pkg/gen` | 代码生成（`landc gen api/service/dao/all <name>`） |
| `pkg/trace` | 链路追踪 |
| `pkg/meta` | 元数据（Meta Tag 解析） |
| `pkg/bootstrap` | 应用生命周期管理 |
| `pkg/cmd` | CLI 命令行框架 |

## 安全行为

### JWT（`pkg/auth`）

- 支持签名算法：**HS256 / RS256 / ES256**，签发时按 `SigningMethod` 选择，验签固定 `WithValidMethods` 算法白名单（防算法混淆攻击）
- **HS256 密钥长度 ≥ 32 字符**（`ValidateSecret` 强制校验，不满足直接拒绝初始化）
- **私钥文件权限强制 0600**（RS256/ES256 加载时校验，过宽权限拒绝）
- 非对称模式支持 `PrivateKeyPath`/`PublicKeyPath` 文件或 `PrivateKey`/`PublicKey` 对象注入

### CORS（`pkg/middleware.CORS`）

- 默认 `Access-Control-Allow-Origin: *`（**不含凭据**）；`AllowCredentials` 开启时必须显式传入来源列表，禁止 `*` + 凭据组合

### WebSocket（`pkg/websocket`）

- 默认启用**同源校验**（`defaultCheckOrigin` 拒绝跨域握手）；生产可显式配置 `CheckOrigin` 定制允许来源

## CLI 工具

```bash
# 初始化项目
landc init myapp

# 代码生成（生成 api/service/dao/model 四层代码）
landc gen all User
landc gen api User
landc gen service User
landc gen dao User
```

## 参数绑定规则

| Tag | 来源 | 示例 |
|-----|------|------|
| `source:"query"` | URL Query 参数 | `/users?page=1` |
| `source:"path"` | URL Path 参数 | `/users/:id` |
| `source:"header"` | HTTP Header | `Authorization` |
| `source:"body"` 或不写 | JSON Body | `{"name":"..."}` |
| `d:"默认值"` | 零值时自动填充 | `d:"1"` |
