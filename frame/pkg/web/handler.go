package web

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"

	"github.com/LandcLi/landc-go/api/core"
	"github.com/gin-gonic/gin"
)

func createHandler(instanceValue reflect.Value, method reflect.Method) gin.HandlerFunc {
	return func(c *gin.Context) {
		// panic recovery 防止反射调用崩溃导致服务器退出
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				// 仅记录日志，不返回堆栈信息到客户端（防止信息泄露）
				fmt.Printf("panic recovered in handler: %v\n%s", r, stack)
				c.JSON(500, gin.H{
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
				c.JSON(400, gin.H{
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
						c.JSON(httpStatusFromCode(apiErr.Code), gin.H{
							"code":    apiErr.Code,
							"message": apiErr.Message,
						})
						return
					}
					// 普通 error：统一 500，不透传内部错误细节
					c.JSON(500, gin.H{
						"code":    50000,
						"message": "internal server error",
					})
					return
				}
			}

			if len(results) > 1 {
				data := results[0].Interface()
				c.JSON(200, gin.H{
					"code":    10000,
					"message": "success",
					"data":    data,
				})
				return
			} else if len(results) == 1 && !lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
				data := results[0].Interface()
				c.JSON(200, gin.H{
					"code":    10000,
					"message": "success",
					"data":    data,
				})
				return
			}
		}

		c.JSON(200, gin.H{
			"code":    10000,
			"message": "success",
		})
	}
}
