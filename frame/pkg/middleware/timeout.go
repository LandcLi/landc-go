package middleware

import (
	"time"

	ginmw "github.com/LandcLi/landc-go/api/middleware/gin"
	"github.com/gin-gonic/gin"
)

// Timeout 为每个请求设置超时上下文。
// 实际实现在 api/middleware/gin，此处为向后兼容的 re-export。
//
//	engine.Use(middleware.Timeout(30 * time.Second))
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return ginmw.Timeout(timeout)
}
