package web

import (
	"fmt"
	"reflect"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func createHandler(instanceValue reflect.Value, method reflect.Method) gin.HandlerFunc {
	return func(c *gin.Context) {
		// panic recovery 防止反射调用崩溃导致服务器退出
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				c.JSON(500, gin.H{
					"error": fmt.Sprintf("internal server error: %v", r),
					"stack": stack,
				})
				c.Abort()
			}
		}()

		methodType := method.Type

		var args []reflect.Value
		args = append(args, instanceValue)

		hasContext := false
		paramIndex := 1

		if methodType.NumIn() > 1 {
			firstParamType := methodType.In(1)
			if firstParamType.String() == "*web.LandcContext" {
				hasContext = true
				paramIndex = 2
			}
		}

		if hasContext {
			args = append(args, reflect.ValueOf(newLandcContext(c)))
		}

		if methodType.NumIn() > paramIndex {
			paramType := methodType.In(paramIndex)
			params, err := parseParamsFromContext(c, paramType)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			args = append(args, params...)
		}

		results := method.Func.Call(args)

		if len(results) > 0 {
			lastResult := results[len(results)-1]
			if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
				if err, _ := lastResult.Interface().(error); err != nil {
					c.JSON(500, gin.H{"error": err.Error()})
					return
				}
			}

			if len(results) > 1 {
				data := results[0].Interface()
				c.JSON(200, data)
				return
			} else if len(results) == 1 && !lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
				data := results[0].Interface()
				c.JSON(200, data)
				return
			}
		}

		c.JSON(200, gin.H{"success": true})
	}
}
