# landc-go Demo

基于 landc-go 的完整可运行示例，覆盖框架核心能力：**Meta Tag 路由 → 参数绑定 → 分层架构（API/Controller/Service/DAO）→ DI → 数据库 → 缓存 → JWT 认证 → 方法级中间件 → 统一响应**。

零外部依赖即可跑通（SQLite + 本地缓存回退）。

## 快速开始

```bash
cd examples/demo
go run .
```

启动后（`config.yaml` 自动加载，默认 `:8080`）：

```bash
# 健康检查（默认路由）
curl http://localhost:8080/health

# 打招呼：首次呼叫写入 SQLite，再次呼叫命中缓存（hit: true）
curl -X POST http://localhost:8080/api/hello \
  -H "Content-Type: application/json" \
  -d '{"name":"landc"}'
# 第一次：{"code":10000,"message":"success","data":{"message":"Hello, landc! (id=1)","id":1,"hit":false}}
# 第二次：{"code":10000,"message":"success","data":{"message":"Hello, landc! (id=1)","id":1,"hit":true}}

# 登录签发 JWT（演示账号：admin/admin123、landc/landc123）
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
# {"code":10000,"message":"success","data":{"token":"eyJhbGciOi... "}}

# 携带 JWT 访问受保护接口（middleware.Auth 方法级中间件）
TOKEN="<上一步的 token>"
curl http://localhost:8080/api/auth/profile -H "Authorization: Bearer $TOKEN"
# {"code":10000,"message":"success","data":{"user_id":10001,"username":"admin","role":"admin"}}

# 不带 token 访问 -> 401
curl http://localhost:8080/api/auth/profile
```

## 演示点

| 能力 | 位置 |
|------|------|
| Meta Tag 路由 / 参数绑定 / `binding` 校验 | `api/hello/v1/say_hello.go`、`api/auth/v1/auth.go` |
| 分层架构 + DI（Gateway/Provide/Require） | `api/*` 接口 + `internal/*_impl` 实现，`internal/impl.go` 触发注册 |
| 数据库读写（`db.GetTx`，SQLite） | `internal/dao_impl/hello/hello.go` |
| 缓存优先（`cache.GetCacheFrom`，Redis 不可用自动回退本地） | `internal/service_impl/hello/hello.go` |
| JWT 签发 / 解析 | `internal/service_impl/auth/auth.go`（`auth.GenerateToken`） |
| 方法级中间件（`WithMethodMiddleware` + `middleware.Auth`） | `internal/cmd/cmd.go` |
| 注入 `*gin.Context` 读取认证用户信息 | `internal/controller/auth/auth.go`（Profile） |
| 统一响应 / 自动建表 / 优雅停机 | 框架内建（`web` + `db.AutoMigrate` + `cmd.Main.Run`） |

## 结构

```
examples/demo/
├── main.go                     # 入口：cmd.Main.Run(ctx)
├── config.yaml                 # SQLite + 本地缓存，零外部依赖
├── go.work                     # 本地联调（use 仓库内各模块）
├── api/                        # 接口定义（Controller 接口 + v1 请求/响应）
│   ├── hello/
│   └── auth/
├── service/                    # 业务接口
├── dao/                        # 数据访问接口
├── model/                      # 数据模型
└── internal/
    ├── cmd/cmd.go              # 主命令：建表 + 注册控制器 + 中间件
    ├── controller/             # Controller 实现
    ├── service_impl/           # Service 实现
    ├── dao_impl/               # DAO 实现
    └── impl.go                 # 触发 DI 注册链
```

## 独立使用（脱离 landc-go 仓库）

demo 内置 `go.work` 指向仓库本地模块（方便随仓库联调）。要独立使用：

1. 删除 `go.work`
2. 修改 `go.mod`：`require github.com/LandcLi/landc-go/frame v0.7.0`（线上版本）
3. `go mod tidy && go run .`
