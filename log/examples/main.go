// log 模块使用示例
//
// 运行：go run ./examples
package main

import (
	"github.com/LandcLi/landc-go/log/facade"
	_ "github.com/LandcLi/landc-go/log/providers/zap"
)

func main() {
	// 1. 默认全局 logger（Console 后端），开箱即用
	facade.Info("server starting", facade.Field{Key: "port", Value: "8080"})
	facade.Debugf("debug detail: %d retries", 3)
	facade.Warn("disk space low", facade.Field{Key: "usage", Value: "85%"})
	facade.Error("request failed", facade.Field{Key: "status", Value: 500})

	// 2. 指定后端（zap）创建独立 logger（zap provider 通过 init 自动注册，名称 "zap"）
	logger := facade.GetLoggerWithProvider("app", "zap")
	logger.Info("zap backend in use", facade.Field{Key: "module", Value: "example"})

	// 3. 命名子 logger（自动继承全局配置，可单独调级别）
	sub := facade.GetLoggerWithName("http-server")
	sub.Infof("handling request %s", "/api/v1/health")

	// 4. 结构化字段链式写入
	ctxLogger := facade.GetLogger().WithField("trace_id", "abc-123").WithField("user_id", 42)
	ctxLogger.Info("user action logged")
}
