package web

import "github.com/gin-gonic/gin"

type LandcContext struct {
	ctx *gin.Context
}

func newLandcContext(ctx *gin.Context) *LandcContext {
	return &LandcContext{ctx: ctx}
}

func (lc *LandcContext) GetHeader(key string) string {
	return lc.ctx.GetHeader(key)
}

func (lc *LandcContext) SetHeader(key, value string) {
	lc.ctx.Header(key, value)
}

func (lc *LandcContext) GetQuery(key string) string {
	return lc.ctx.Query(key)
}

func (lc *LandcContext) GetQueryArray(key string) []string {
	return lc.ctx.QueryArray(key)
}

func (lc *LandcContext) GetParam(key string) string {
	return lc.ctx.Param(key)
}

func (lc *LandcContext) ClientIP() string {
	return lc.ctx.ClientIP()
}

func (lc *LandcContext) Method() string {
	return lc.ctx.Request.Method
}

func (lc *LandcContext) Path() string {
	return lc.ctx.Request.URL.Path
}

func (lc *LandcContext) GetGinContext() *gin.Context {
	return lc.ctx
}
