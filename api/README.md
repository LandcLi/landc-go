# landc-go/api

API 规范模块，定义统一的响应结构、错误码体系和框架中间件。

## 安装

```bash
go get github.com/LandcLi/landc-go/api
```

## 核心概念

### 统一响应结构

```go
type Response struct {
    Code      int         `json:"code"`
    Message   string      `json:"message"`
    Data      interface{} `json:"data,omitempty"`
    Details   interface{} `json:"details,omitempty"`
    TraceID   string      `json:"trace_id,omitempty"`
    Latency   int64       `json:"latency,omitempty"`
    Timestamp time.Time   `json:"timestamp"`
}
```

### 错误码体系

| 范围 | 含义 |
|------|------|
| 10000 | 成功 |
| 40000-40099 | 客户端错误（参数/验证） |
| 40100 | 未认证 |
| 40300 | 无权限 |
| 40400 | 资源不存在 |
| 50000-50099 | 服务端错误 |
| 60000-99999 | 自定义业务错误 |

### 使用

```go
import "github.com/LandcLi/landc-go/api/core"

// 创建错误
err := core.NewError(core.ErrorCodeBadRequest, "参数不合法")

// 预定义错误（返回副本，不可变）
err := core.ErrBadRequest()
err := core.ErrUnauthorized()

// 自定义业务错误（code 必须在 60000-99999）
err, _ := core.NewCustomError(60001, "余额不足")
```

## 框架中间件

- `middleware/gin` — Gin 中间件（自动转换错误为规范响应）
- `middleware/goframe` — GoFrame 中间件（**可选集成**，默认不编译；使用 `go build -tags goframe` 启用，避免引入 GoFrame 全量依赖）
