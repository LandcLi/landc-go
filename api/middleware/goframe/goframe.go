package goframe

import (
	"net/http"
	"time"

	"github.com/LandcLi/landc-go/api/core"
	"github.com/LandcLi/landc-go/api/trace"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/net/gtrace"
)

// GoFrameMiddlewareConfig GoFrame中间件配置
type GoFrameMiddlewareConfig struct {
	ErrorConverter  *GoFrameErrorConverter
	ResponseBuilder *core.ResponseBuilder
	HeaderProcessor *core.HeaderProcessor
	EnableHeaders   bool
}

// NewGoFrameMiddlewareConfig 创建默认的GoFrame中间件配置
func NewGoFrameMiddlewareConfig() *GoFrameMiddlewareConfig {
	return &GoFrameMiddlewareConfig{
		ErrorConverter:  NewGoFrameErrorConverter(),
		ResponseBuilder: core.NewResponseBuilder(),
		HeaderProcessor: core.NewHeaderProcessor(),
		EnableHeaders:   true,
	}
}

// WithErrorConverter 设置错误转换器
func (config *GoFrameMiddlewareConfig) WithErrorConverter(converter *GoFrameErrorConverter) *GoFrameMiddlewareConfig {
	config.ErrorConverter = converter
	return config
}

// WithResponseBuilder 设置响应构造器
func (config *GoFrameMiddlewareConfig) WithResponseBuilder(builder *core.ResponseBuilder) *GoFrameMiddlewareConfig {
	config.ResponseBuilder = builder
	return config
}

// WithHeaderProcessor 设置请求头处理器
func (config *GoFrameMiddlewareConfig) WithHeaderProcessor(processor *core.HeaderProcessor) *GoFrameMiddlewareConfig {
	config.HeaderProcessor = processor
	return config
}

// WithEnableHeaders 设置是否启用请求头处理
func (config *GoFrameMiddlewareConfig) WithEnableHeaders(enable bool) *GoFrameMiddlewareConfig {
	config.EnableHeaders = enable
	return config
}

// GoFrameMiddleware 返回使用默认配置的GoFrame中间件
func GoFrameMiddleware() ghttp.HandlerFunc {
	return GoFrameMiddlewareWithConfig(NewGoFrameMiddlewareConfig())
}

// GoFrameMiddlewareWithConfig 返回使用指定配置的GoFrame中间件
func GoFrameMiddlewareWithConfig(config *GoFrameMiddlewareConfig) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		start := time.Now()

		// 提取链路跟踪信息
		t := trace.ExtractFromRequest(r.Request)

		ctx, err := gtrace.WithTraceID(r.Context(), t.TraceID)
		if err != nil {
			r.Response.WriteHeader(http.StatusInternalServerError)
			r.Response.WriteJson(&core.Response{
				Code:    int(core.CodeError),
				Message: "internal server error",
				TraceID: t.TraceID,
			})
			return
		}
		r.SetCtx(ctx)

		// 处理请求头
		if config.EnableHeaders {
			headers := config.HeaderProcessor.Process(r.Request, r.Response.Writer)
			r.SetCtxVar("request_headers", headers)
		}

		// 执行后续 handler
		r.Middleware.Next()

		latency := time.Since(start)

		// 构建规范响应
		resp := buildGoFrameResponse(r, t, latency, config)

		// 清除并写入最终响应
		r.Response.ClearBuffer()
		r.Response.WriteJson(resp)
	}
}

func buildGoFrameResponse(r *ghttp.Request, t *trace.Trace, latency time.Duration, config *GoFrameMiddlewareConfig) *core.Response {
	// 检查错误
	if err := r.GetError(); err != nil {
		apiErr := config.ErrorConverter.ConvertGoFrameError(err)
		return config.ResponseBuilder.BuildErrorResponse(apiErr, t.TraceID, latency)
	}

	// 获取 handler 响应数据
	data := r.GetHandlerResponse()

	if resp, ok := data.(*core.Response); ok {
		resp.TraceID = t.TraceID
		resp.Latency = latency.Milliseconds()
		if resp.Timestamp.IsZero() {
			resp.Timestamp = time.Now()
		}
		return resp
	}

	return config.ResponseBuilder.BuildSuccessResponse(data, t.TraceID, latency)
}
