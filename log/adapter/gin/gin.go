package gin

import (
	"fmt"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/LandcLi/landc-go/log/internal/logger"
)

// GinLogger 是gin框架的日志适配器
type GinLogger struct {
	log logger.Logger
}

// NewGinLogger 创建一个新的gin日志适配器
func NewGinLogger(log logger.Logger) *GinLogger {
	return &GinLogger{
		log: log,
	}
}

// Logger 返回gin的日志中间件
func (g *GinLogger) Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 处理请求
		c.Next()

		// 结束时间
		endTime := time.Now()

		// 执行时间
		latencyTime := endTime.Sub(startTime)

		// 请求方式
		reqMethod := c.Request.Method

		// 请求路由
		reqUri := c.Request.RequestURI

		// 状态码
		statusCode := c.Writer.Status()

		// 请求IP
		clientIP := c.ClientIP()
		traceID := c.Request.Header.Get("X-Trace-ID")
		if traceID == "" {
			var ok bool
			traceID, ok = c.Request.Context().Value("trace_id").(string)
			if !ok {
				traceID = fmt.Sprintf("%d", time.Now().UnixNano())
			}
		}

		// 日志字段
		fields := []logger.Field{
			{Key: "status", Value: statusCode},
			{Key: "method", Value: reqMethod},
			{Key: "uri", Value: reqUri},
			{Key: "ip", Value: clientIP},
			{Key: "latency", Value: latencyTime},
			{Key: "timestamp", Value: endTime},
			{Key: "trace_id", Value: traceID},
			{Key: "error", Value: c.Errors},
		}

		// 根据状态码设置日志级别
		switch {
		case statusCode >= 500:
			g.log.Error("", fields...)
		case statusCode >= 400:
			g.log.Warn("", fields...)
		default:
			g.log.Info("", fields...)
		}
	}
}

// Recovery 返回gin的恢复中间件，使用我们的日志门面记录错误
func (g *GinLogger) Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 收集 stack trace
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				stackTrace := string(buf[:n])

				// 记录错误日志（包含 stack trace）
				g.log.Error("panic recovered",
					logger.Field{Key: "method", Value: c.Request.Method},
					logger.Field{Key: "uri", Value: c.Request.RequestURI},
					logger.Field{Key: "ip", Value: c.ClientIP()},
					logger.Field{Key: "error", Value: err},
					logger.Field{Key: "stacktrace", Value: stackTrace},
				)

				// 响应500错误
				c.AbortWithStatus(500)
			}
		}()

		c.Next()
	}
}

// UseWithGin 将日志适配器应用到gin引擎
func UseWithGin(r *gin.Engine, log logger.Logger) {
	ginLogger := NewGinLogger(log)
	r.Use(ginLogger.Logger())
	r.Use(ginLogger.Recovery())
}
