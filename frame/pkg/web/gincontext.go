package web

import (
	"context"

	"github.com/gin-gonic/gin"
)
// GinContextFromContext 从 context.Context 中还原 *gin.Context
// 如果 ctx 本身是 *gin.Context，也支持直接类型断言
func GinContextFromContext(ctx context.Context) (*gin.Context, bool) {
	if gc, ok := ctx.(*gin.Context); ok {
		return gc, true
	}
	return nil, false
}
