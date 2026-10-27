package web

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"

	"github.com/LandcLi/landc-go/api/core"
	"github.com/LandcLi/landc-go/frame/pkg/trace"
	"github.com/LandcLi/landc-go/tools/generate"
	"github.com/gin-gonic/gin"
)

// ensureTraceID 确保当前请求有一个 trace_id。
//
// 已由中间件（middleware.Trace / GinMiddleware）设置时直接复用；
// 否则生成一个并写回 gin.Context 与响应头。
//
// 为什么需要兜底：这套框架里 trace_id 只由 trace 类中间件产生，而
// web.NewServer 并不默认挂它们。结果是「配了链路追踪」的服务里，
// 只有挂了 GinMiddleware 的那个才有 trace_id —— 同一个系统里有的接口带
// 有的不带，排查慢请求时根本拿不到标识去串日志。
func ensureTraceID(c *gin.Context) string {
	if id := c.GetString("trace_id"); id != "" {
		// 中间件已设置，但可能只写进了 gin.Context。补一份到 request context，
		// 让下游的 ProxyClient 等组件能从标准 context.Context 里取到。
		if trace.TraceID(c.Request.Context()) == "" {
			c.Request = c.Request.WithContext(trace.WithTraceID(c.Request.Context(), id))
		}
		return id
	}
	id := generate.UUID()
	c.Set("trace_id", id)
	// 同步写入 request context：跨服务调用（di.ProxyClient）从这里取 trace_id
	// 透传给下游，否则一次请求在 A 服务的日志和 B 服务的日志里对不上号。
	c.Request = c.Request.WithContext(trace.WithTraceID(c.Request.Context(), id))
	// 回写响应头，方便调用方/网关在 body 之外也能拿到
	c.Header("X-Trace-ID", id)
	return id
}

// respondJSON 统一写出 JSON 响应，并保证带上 trace_id。
func respondJSON(c *gin.Context, status int, payload gin.H) {
	if payload == nil {
		payload = gin.H{}
	}
	if _, ok := payload["trace_id"]; !ok {
		payload["trace_id"] = ensureTraceID(c)
	}
	c.JSON(status, payload)
}

func createHandler(instanceValue reflect.Value, method reflect.Method) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 请求最开头就确定 trace_id：后续任何分支（含 panic）都能带上它
		ensureTraceID(c)

		// panic recovery 防止反射调用崩溃导致服务器退出
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				// 仅记录日志，不返回堆栈信息到客户端（防止信息泄露）
				fmt.Printf("panic recovered in handler: %v\n%s", r, stack)
				respondJSON(c, 500, gin.H{
					"code":    50000,
					"message": "internal server error",
				})
				c.Abort()
			}
		}()

		methodType := method.Type

		var args []reflect.Value
		args = append(args, instanceValue)

		paramIndex := 1

		// 检查方法第一个参数是否是 context.Context 接口
		if methodType.NumIn() > 1 {
			firstParamType := methodType.In(1)
			ginContextType := reflect.TypeOf((*gin.Context)(nil))
			contextInterface := reflect.TypeOf((*context.Context)(nil)).Elem()
			switch {
			case firstParamType == ginContextType:
				// 显式 *gin.Context：传入 gin 上下文（保留 gin API 能力）
				paramIndex = 2
				args = append(args, reflect.ValueOf(c))
			case firstParamType.Implements(contextInterface):
				// 任意 context.Context 接口：传入携带资源作用域的请求 ctx
				// （c.Request.Context() 由资源作用域中间件写入 Scope，标准 Go 风格）
				paramIndex = 2
				args = append(args, reflect.ValueOf(c.Request.Context()))
			}
		}

		if methodType.NumIn() > paramIndex {
			paramType := methodType.In(paramIndex)
			params, err := parseParamsFromContext(c, paramType)
			if err != nil {
				respondJSON(c, 400, gin.H{
					"code":    40000,
					"message": err.Error(),
				})
				return
			}
			args = append(args, params...)
		}

		results := method.Func.Call(args)

		// 若业务层已通过 Abort 写入响应（如中间件/授权失败返回 4xx），
		// 不再覆盖响应，避免双重写入。
		if c.IsAborted() {
			return
		}

		if len(results) > 0 {
			lastResult := results[len(results)-1]
			if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
				if err, _ := lastResult.Interface().(error); err != nil {
					// 记录完整错误日志（含包装链与 cause），不向客户端透传内部细节
					fmt.Printf("handler %s.%s returned error: %v\n", instanceValue.Type(), method.Name, err)
					// 业务错误码透传：*core.Error（含 %w 包装链）→ 响应携带 code + message，
					// HTTP 状态码按错误码映射；Details/Cause 不进响应体（防泄露，日志已记录）
					var apiErr *core.Error
					if errors.As(err, &apiErr) {
						respondJSON(c, httpStatusFromCode(apiErr.Code), gin.H{
							"code":    apiErr.Code,
							"message": apiErr.Message,
						})
						return
					}
					// 普通 error：统一 500，不透传内部错误细节
					respondJSON(c, 500, gin.H{
						"code":    50000,
						"message": "internal server error",
					})
					return
				}
			}

			if len(results) > 1 {
				data := results[0].Interface()
				respondJSON(c, 200, gin.H{
					"code":    10000,
					"message": "success",
					"data":    data,
				})
				return
			} else if len(results) == 1 && !lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
				data := results[0].Interface()
				respondJSON(c, 200, gin.H{
					"code":    10000,
					"message": "success",
					"data":    data,
				})
				return
			}
		}

		respondJSON(c, 200, gin.H{
			"code":    10000,
			"message": "success",
		})
	}
}
