# landc-go/log

日志门面模块，提供统一的日志接口，支持多种后端实现。

## 安装

```bash
go get github.com/LandcLi/landc-go/log
```

## 使用

```go
import "github.com/LandcLi/landc-go/log/facade"

// 直接使用全局日志
facade.Info("server started", facade.Field{Key: "port", Value: 8080})
facade.Debugf("request: %s %s", method, path)
facade.Error("failed", facade.Field{Key: "err", Value: err})

// 获取 logger 实例（支持链式调用）
log := facade.GetLogger()
log.WithFields(
    facade.Field{Key: "user_id", Value: 123},
    facade.Field{Key: "action", Value: "login"},
).Info("user logged in")
```

## 支持的后端

| 后端 | Provider 名称 | 引入方式 |
|------|-------------|---------|
| Console（默认） | `console` | 内置 |
| 标准库 log | `std` | 内置 |
| Zap | `zap` | `import _ "github.com/LandcLi/landc-go/log/providers/zap"` |
| Logrus | `logrus` | `import _ "github.com/LandcLi/landc-go/log/providers/logrus"` |

## 配置

```go
// 方式一：通过 Option 配置
log := facade.GetLoggerWithProvider("app", "zap",
    facade.WithLevel(facade.InfoLevel),
    facade.WithFormat("json"),
    facade.WithOutputPath("./logs/app.log"),
    facade.WithMaxLogSize(100),      // 100MB 切割
    facade.WithMaxLogFiles(10),      // 保留 10 个文件
    facade.WithCompressLogs(true),   // 压缩旧日志
)

// 方式二：通过 LogConfig 配置
config := facade.NewLogConfig()
config.Level = "info"
config.Format = "json"
config.Output = "./logs/app.log"
log := facade.GetLoggerWithLogConfig(config)
```

## 框架适配

- **Gin**：`log/adapter/gin` — 替换 Gin 默认日志
- **GoFrame**：`log/adapter/gf` — 实现 glog.ILogger 接口
