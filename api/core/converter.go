package core

import (
	"time"
)

// ErrorConverter 错误转换接口
type ErrorConverter interface {
	ConvertError(err error) *Error
	ConvertResponse(data interface{}) *Response
}

// DefaultErrorConverter 默认的错误转换器
type DefaultErrorConverter struct{}

// NewDefaultErrorConverter 创建默认的错误转换器
func NewDefaultErrorConverter() *DefaultErrorConverter {
	return &DefaultErrorConverter{}
}

// ConvertError 转换错误
func (c *DefaultErrorConverter) ConvertError(err error) *Error {
	return ConvertFrameworkError(err)
}

// ConvertResponse 转换响应
func (c *DefaultErrorConverter) ConvertResponse(data interface{}) *Response {
	if resp, ok := data.(*Response); ok {
		return resp
	}

	return &Response{
		Code:      int(CodeSuccess),
		Message:   "success",
		Data:      data,
		Timestamp: time.Now(),
	}
}

// ResponseBuilder 响应构造器
type ResponseBuilder struct {
	errorConverter  ErrorConverter
	responseBuilder func(data interface{}) *Response
	errorBuilder    func(err error) *Error
}

// NewResponseBuilder 创建响应构造器
func NewResponseBuilder() *ResponseBuilder {
	return &ResponseBuilder{
		errorConverter: NewDefaultErrorConverter(),
	}
}

// WithErrorConverter 设置错误转换器
func (b *ResponseBuilder) WithErrorConverter(converter ErrorConverter) *ResponseBuilder {
	b.errorConverter = converter
	return b
}

// WithResponseBuilder 设置自定义响应构造函数
func (b *ResponseBuilder) WithResponseBuilder(builder func(data interface{}) *Response) *ResponseBuilder {
	b.responseBuilder = builder
	return b
}

// WithErrorBuilder 设置自定义错误构造函数
func (b *ResponseBuilder) WithErrorBuilder(builder func(err error) *Error) *ResponseBuilder {
	b.errorBuilder = builder
	return b
}

// BuildResponse 构建响应
func (b *ResponseBuilder) BuildResponse(data interface{}) *Response {
	if b.responseBuilder != nil {
		return b.responseBuilder(data)
	}
	return b.errorConverter.ConvertResponse(data)
}

// BuildError 构建错误
func (b *ResponseBuilder) BuildError(err error) *Error {
	if b.errorBuilder != nil {
		return b.errorBuilder(err)
	}
	return b.errorConverter.ConvertError(err)
}

// BuildErrorResponse 构建错误响应
func (b *ResponseBuilder) BuildErrorResponse(err error, traceID string, latency time.Duration) *Response {
	apiErr := b.BuildError(err)
	if apiErr == nil {
		apiErr = NewError(CodeSuccess, "success")
	}

	return &Response{
		Code:      int(apiErr.Code),
		Message:   apiErr.Message,
		Details:   apiErr.Details,
		TraceID:   traceID,
		Latency:   latency.Milliseconds(),
		Timestamp: time.Now(),
	}
}

// BuildSuccessResponse 构建成功响应
func (b *ResponseBuilder) BuildSuccessResponse(data interface{}, traceID string, latency time.Duration) *Response {
	resp := b.BuildResponse(data)
	resp.TraceID = traceID
	resp.Latency = latency.Milliseconds()
	if resp.Timestamp.IsZero() {
		resp.Timestamp = time.Now()
	}
	return resp
}
