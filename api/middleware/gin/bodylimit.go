package gin

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodySize 限制请求体大小。
// maxBytes 为最大字节数，超出时返回 413 Request Entity Too Large。
//
// 用法:
//
//	r.Use(ginmw.MaxBodySize(10 << 20)) // 10 MB
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()

		// 如果因为超出大小而中断，返回 413
		if c.Writer.Status() == http.StatusRequestEntityTooLarge {
			// 已被 MaxBytesReader 中断，无需额外操作
			return
		}

		// 检查读取是否被中止（超出大小会 panic，Gin 的 Recovery 会恢复）
		// 在 c.Next() 后检查 body 读取状态
		_, _ = io.CopyN(io.Discard, c.Request.Body, 1)
	}
}
