package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/LandcLi/landc-go/api/core"
	"github.com/LandcLi/landc-go/frame/pkg/auth"
	"github.com/LandcLi/landc-go/frame/pkg/response"
	"github.com/LandcLi/landc-go/frame/pkg/trace"
	"github.com/LandcLi/landc-go/tools/generate"
	"github.com/LandcLi/landc-go/log/facade"
	"github.com/gin-gonic/gin"
)

// Logger 日志中间件（使用 landc-go/log 结构化日志）
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Set("_start_time", start)

		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []facade.Field{
			{Key: "status", Value: status},
			{Key: "method", Value: c.Request.Method},
			{Key: "path", Value: path},
			{Key: "query", Value: query},
			{Key: "ip", Value: c.ClientIP()},
			{Key: "latency_ms", Value: latency.Milliseconds()},
			{Key: "trace_id", Value: c.GetString("trace_id")},
			{Key: "user_agent", Value: c.Request.UserAgent()},
		}

		if len(c.Errors) > 0 {
			fields = append(fields, facade.Field{Key: "errors", Value: c.Errors.String()})
		}

		switch {
		case status >= 500:
			facade.GetLogger().WithFields(fields...).Error("server error")
		case status >= 400:
			facade.GetLogger().WithFields(fields...).Warn("client error")
		default:
			facade.GetLogger().WithFields(fields...).Info("request completed")
		}
	}
}

// Recovery panic 恢复中间件（使用 landc-go/log 记录 + landc-go/api 错误码响应）
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())

				facade.GetLogger().WithFields(
					facade.Field{Key: "error", Value: fmt.Sprintf("%v", r)},
					facade.Field{Key: "stack", Value: stack},
					facade.Field{Key: "path", Value: c.Request.URL.Path},
					facade.Field{Key: "method", Value: c.Request.Method},
					facade.Field{Key: "trace_id", Value: c.GetString("trace_id")},
				).Error("panic recovered")

				response.InternalServerError(c, "internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// CORS 跨域中间件
func CORS(allowOrigins ...string) gin.HandlerFunc {
	origin := "*"
	if len(allowOrigins) > 0 {
		origin = strings.Join(allowOrigins, ",")
	}

	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Trace-ID, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Trace 链路追踪中间件（使用 landc-go/tools/generate 生成 UUID）
func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = c.GetHeader("X-Request-ID")
		}
		if traceID == "" {
			traceID = generate.UUID()
		}

		// 注入到 context（供 trace 包使用）
		ctx := trace.WithTraceID(c.Request.Context(), traceID)
		ctx = trace.WithSpanID(ctx, generate.UUID())
		c.Request = c.Request.WithContext(ctx)

		// 同时存入 gin.Context（供 response 包使用）
		c.Set("trace_id", traceID)
		c.Header("X-Trace-ID", traceID)

		c.Next()
	}
}

// Auth JWT 认证中间件（使用 landc-go/api 错误码）
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "authorization header is required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "authorization header format is invalid")
			c.Abort()
			return
		}

		claims, err := auth.ParseToken(parts[1])
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("client_id", claims.ClientID)
		c.Set("scope", claims.Scope)
		c.Set("ip", claims.IP)
		c.Set("device_fingerprint", claims.DeviceFingerprint)
		c.Set("token_type", claims.TokenType)
		c.Next()
	}
}

// RoleRequired 角色验证中间件
func RoleRequired(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetString("role")
		if userRole == "" {
			response.Forbidden(c, "no role found")
			c.Abort()
			return
		}

		for _, role := range roles {
			if userRole == role {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "insufficient permissions")
		c.Abort()
	}
}

// ErrorHandler 统一错误处理中间件（将 gin.Error 转换为 landc-go/api 规范响应）
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 如果已经写过响应，跳过
		if c.Writer.Written() {
			return
		}

		if len(c.Errors) > 0 {
			lastErr := c.Errors.Last().Err

			// 如果是 landc-go/api 的 Error 类型，直接使用其错误码
			if apiErr, ok := lastErr.(*core.Error); ok {
				response.ErrorFromCoreError(c, apiErr)
				return
			}

			// 普通错误
			response.InternalServerError(c, lastErr.Error())
		}
	}
}
