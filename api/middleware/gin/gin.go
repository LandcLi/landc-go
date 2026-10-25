package gin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/LandcLi/landc-go/api/core"
	"github.com/LandcLi/landc-go/api/trace"

	"github.com/gin-gonic/gin"
)

// GinMiddlewareConfig Gin中间件配置
type GinMiddlewareConfig struct {
	ErrorConverter    *GinErrorConverter
	ResponseBuilder   *core.ResponseBuilder
	HeaderProcessor   *core.HeaderProcessor
	EnableHeaders     bool
	CustomErrorParser func(c *gin.Context, body []byte) error
}

// NewGinMiddlewareConfig 创建默认的Gin中间件配置
func NewGinMiddlewareConfig() *GinMiddlewareConfig {
	return &GinMiddlewareConfig{
		ErrorConverter:    NewGinErrorConverter(),
		ResponseBuilder:   core.NewResponseBuilder(),
		HeaderProcessor:   core.NewHeaderProcessor(),
		EnableHeaders:     true,
		CustomErrorParser: nil,
	}
}

// WithErrorConverter 设置错误转换器
func (config *GinMiddlewareConfig) WithErrorConverter(converter *GinErrorConverter) *GinMiddlewareConfig {
	config.ErrorConverter = converter
	return config
}

// WithResponseBuilder 设置响应构造器
func (config *GinMiddlewareConfig) WithResponseBuilder(builder *core.ResponseBuilder) *GinMiddlewareConfig {
	config.ResponseBuilder = builder
	return config
}

// WithHeaderProcessor 设置请求头处理器
func (config *GinMiddlewareConfig) WithHeaderProcessor(processor *core.HeaderProcessor) *GinMiddlewareConfig {
	config.HeaderProcessor = processor
	return config
}

// WithEnableHeaders 设置是否启用请求头处理
func (config *GinMiddlewareConfig) WithEnableHeaders(enable bool) *GinMiddlewareConfig {
	config.EnableHeaders = enable
	return config
}

// WithCustomErrorParser 设置自定义错误解析器
// body 参数是 handler 写入的原始响应体字节
func (config *GinMiddlewareConfig) WithCustomErrorParser(parser func(c *gin.Context, body []byte) error) *GinMiddlewareConfig {
	config.CustomErrorParser = parser
	return config
}

// GinMiddleware 返回使用默认配置的Gin中间件
func GinMiddleware() gin.HandlerFunc {
	return GinMiddlewareWithConfig(NewGinMiddlewareConfig())
}

// GinMiddlewareWithConfig 返回使用指定配置的Gin中间件
func GinMiddlewareWithConfig(config *GinMiddlewareConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 提取链路跟踪信息
		t := trace.ExtractFromRequest(c.Request)
		c.Set("trace_id", t.TraceID)

		// 处理请求头
		if config.EnableHeaders {
			headers := config.HeaderProcessor.Process(c.Request, c.Writer)
			c.Set("request_headers", headers)
		}

		// 拦截 handler 写入的响应体
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// 执行后续 handler
		c.Next()

		latency := time.Since(start)

		// 构建规范响应
		resp := buildGinResponse(c, blw, t, latency, config)

		// 写入最终响应到真正的 ResponseWriter
		respJSON, err := json.Marshal(resp)
		if err != nil {
			blw.Header().Set("Content-Type", "application/json")
			blw.ResponseWriter.WriteHeader(http.StatusInternalServerError)
			_, _ = blw.ResponseWriter.WriteString(`{"code":50000,"message":"internal server error","trace_id":"` + t.TraceID + `"}`)
			return
		}

		blw.Header().Set("Content-Type", "application/json")
		blw.ResponseWriter.WriteHeader(blw.statusCode())
		// 响应体写入失败时连接已不可用，记录到日志后交由上层处理
		if _, err := blw.ResponseWriter.Write(respJSON); err != nil {
			c.Error(err) //nolint:errcheck // gin 的 Error 仅用于记录，忽略其返回值
		}
	}
}

// bodyLogWriter 拦截响应体的 Writer。
//
// 它同时接管状态码：gin 的 responseWriter 在 WriteHeader 时只记录 status，
// 真正写出去发生在 WriteHeaderNow（由 Write 触发）。因为本类型拦截了 Write，
// 底层写入不会自动发生，若不显式记录 status，最终响应就只能由中间件
// 无条件写 200 —— 这正是历史缺陷：handler 返回 400/500 时 HTTP 状态码仍为
// 200，只有 body 里的业务 code 是对的，网关 / 监控 / 按状态码判断的客户端
// 全部误判为成功。
type bodyLogWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

// Write 拦截写入，存储到 buffer 中
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

// WriteString 拦截字符串写入
func (w *bodyLogWriter) WriteString(s string) (int, error) {
	return w.body.WriteString(s)
}

// WriteHeader 记录 handler 期望的状态码，不立即写到底层。
// 最终状态码由中间件在构建完响应后统一写出。
func (w *bodyLogWriter) WriteHeader(code int) {
	if code > 0 {
		w.status = code
	}
}

// WriteHeaderNow 抑制隐式写出（gin 在 Write 时会调用），保证状态码由中间件决定。
func (w *bodyLogWriter) WriteHeaderNow() {}

// statusCode 返回最终应写出的状态码，未显式设置时为 200。
func (w *bodyLogWriter) statusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func buildGinResponse(c *gin.Context, blw *bodyLogWriter, t *trace.Trace, latency time.Duration, config *GinMiddlewareConfig) *core.Response {
	// 优先检查 Gin 错误链
	if len(c.Errors) > 0 {
		lastErr := c.Errors.Last()
		apiErr := config.ErrorConverter.ConvertGinError(lastErr)
		return config.ResponseBuilder.BuildErrorResponse(apiErr, t.TraceID, latency)
	}

	// 检查自定义错误解析
	if blw.body.Len() > 0 {
		bodyBytes := blw.body.Bytes()

		if config.CustomErrorParser != nil {
			if err := config.CustomErrorParser(c, bodyBytes); err != nil {
				apiErr := config.ErrorConverter.ConvertGinError(err)
				return config.ResponseBuilder.BuildErrorResponse(apiErr, t.TraceID, latency)
			}
		}

		// 尝试解析为 core.Response
		var resp core.Response
		if json.Unmarshal(bodyBytes, &resp) == nil && resp.Code != 0 {
			resp.TraceID = t.TraceID
			resp.Latency = latency.Milliseconds()
			if resp.Timestamp.IsZero() {
				resp.Timestamp = time.Now()
			}
			return &resp
		}

		// 尝试解析为普通 JSON
		var parsedData interface{}
		if json.Unmarshal(bodyBytes, &parsedData) == nil {
			return config.ResponseBuilder.BuildSuccessResponse(parsedData, t.TraceID, latency)
		}

		// 非 JSON，作为字符串数据
		return config.ResponseBuilder.BuildSuccessResponse(blw.body.String(), t.TraceID, latency)
	}

	// 无响应体
	return config.ResponseBuilder.BuildSuccessResponse(nil, t.TraceID, latency)
}
