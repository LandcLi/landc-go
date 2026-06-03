# landc-go

**landc-go** 是一套 Go 语言应用开发框架，采用多 Module 单仓库（Monorepo）架构，各模块可独立引用、独立发版。

## 模块概览

| 模块 | Import 路径 | 说明 | 可独立使用 |
|------|------------|------|-----------|
| **log** | `github.com/LandcLi/landc-go/log` | 日志门面，统一接口支持 Zap / Logrus / Console | ✅ |
| **tools** | `github.com/LandcLi/landc-go/tools` | 通用工具集（缓存 / 字符串 / 数字 / UUID / 地理 / 邮件 / Tag 验证） | ✅ |
| **api** | `github.com/LandcLi/landc-go/api` | API 规范（统一响应结构 / 错误码体系 / 框架中间件） | ✅ |
| **frame** | `github.com/LandcLi/landc-go/frame` | Web 全栈框架（Web / DI / ORM / Redis / JWT / gRPC / WebSocket / ...） | 依赖以上三者 |

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

## 版本管理

每个模块独立发版，使用路径前缀 Git Tag：

```bash
git tag log/v0.1.0 && git push origin log/v0.1.0
git tag tools/v0.1.0 && git push origin tools/v0.1.0
git tag api/v0.1.0 && git push origin api/v0.1.0
git tag frame/v0.1.0 && git push origin frame/v0.1.0
```

## 架构设计

```
依赖方向（单向，无循环）:

    frame  →  api
     │       │
     ├───→  log
     │
     └───→ tools
```

## 目录结构

```
landc-go/
├── README.md
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
│   └── tag/                 结构体 Tag 验证器
├── api/                     # API 规范
│   ├── go.mod               module github.com/LandcLi/landc-go/api
│   ├── core/                Response / ErrorCode / Error
│   ├── middleware/          Gin / GoFrame 中间件
│   └── trace/               链路追踪
└── frame/                   # Web 全栈框架
    ├── go.mod               module github.com/LandcLi/landc-go/frame
    └── pkg/
        ├── web/             Web 引擎（Meta Tag 路由 / 参数绑定 / 文件上传）
        ├── di/              依赖注入容器 + Gateway 服务网关（本地/远程透明切换）
        ├── config/          配置管理（YAML / 环境变量 / 热更新）
        ├── db/              数据库（GORM + 迁移工具）
        ├── cache/           Redis 缓存
        ├── auth/            JWT 认证
        ├── middleware/      中间件（Logger / Recovery / CORS / Auth / 限流 / 熔断）
        ├── response/        统一响应格式
        ├── i18n/            国际化
        ├── session/         Session（Redis / Memory）
        ├── websocket/       WebSocket（Hub / 房间 / 广播）
        ├── cron/            定时任务
        ├── openapi/         OpenAPI 3.0 文档自动生成
        ├── registry/        服务注册 / 发现（etcd）
        ├── grpc/            gRPC 服务端 / 客户端
        ├── gen/             代码生成（landc gen api/service/dao）
        ├── trace/           链路追踪
        ├── meta/            元数据
        ├── bootstrap/       应用生命周期
        └── cmd/             CLI 命令行
```

## 设计理念

- **仿 GoFrame 风格**：Controller 通过 Meta Tag 声明路由，请求参数结构体自动绑定
- **分层架构**：API → Controller → Service → DAO，各层通过接口解耦
- **模块化**：各基础模块可独立使用，不强制绑定完整框架
- **约定优于配置**：合理的默认值，最小化样板代码

## License

MIT
