package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// Timeout 为每个请求设置超时上下文。
// 超时后请求的 Context 会 Done，由业务代码通过 ctx.Err() 或 ctx.Done() 感知。
// 如果超时且尚未写入响应，自动返回 504。
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		if ctx.Err() == context.DeadlineExceeded {
			if !c.Writer.Written() {
				c.AbortWithStatusJSON(504, gin.H{
					"code":    504,
					"message": "request timeout",
				})
			}
		}
	}
}
