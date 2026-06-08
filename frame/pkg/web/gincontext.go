package web

import (
	"context"

	"github.com/gin-gonic/gin"
)

// LandcContext 是 *gin.Context 的类型别名，用于 Controller 方法签名中
// 提供更语义化的上下文参数名称，同时保持与 gin 的完全兼容。
type LandcContext = gin.Context

// GinContextFromContext 从 context.Context 中还原 *gin.Context
// 如果 ctx 本身是 *gin.Context，也支持直接类型断言
func GinContextFromContext(ctx context.Context) (*gin.Context, bool) {
	if gc, ok := ctx.(*gin.Context); ok {
		return gc, true
	}
	return nil, false
}
