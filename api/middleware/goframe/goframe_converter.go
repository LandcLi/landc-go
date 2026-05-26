package goframe

import (
	"github.com/LandcLi/landc-go/api/core"
)

// 编译时接口验证
var _ core.ErrorConverter = (*GoFrameErrorConverter)(nil)

// GoFrameErrorConverter GoFrame框架错误转换器
type GoFrameErrorConverter struct {
	customConverter func(err error) *core.Error
}

// NewGoFrameErrorConverter 创建GoFrame错误转换器
func NewGoFrameErrorConverter() *GoFrameErrorConverter {
	return &GoFrameErrorConverter{}
}

// WithCustomConverter 设置自定义转换器
func (c *GoFrameErrorConverter) WithCustomConverter(converter func(err error) *core.Error) *GoFrameErrorConverter {
	c.customConverter = converter
	return c
}

// ConvertError 实现 ErrorConverter 接口
func (c *GoFrameErrorConverter) ConvertError(err error) *core.Error {
	if err == nil {
		return nil
	}

	if apiErr, ok := err.(*core.Error); ok {
		return apiErr
	}

	if c.customConverter != nil {
		return c.customConverter(err)
	}

	return core.ConvertFrameworkError(err)
}

// ConvertResponse 实现 ErrorConverter 接口
func (c *GoFrameErrorConverter) ConvertResponse(data interface{}) *core.Response {
	converter := core.NewDefaultErrorConverter()
	return converter.ConvertResponse(data)
}

// ConvertGoFrameError 转换 GoFrame 框架特有的错误
func (c *GoFrameErrorConverter) ConvertGoFrameError(err error) *core.Error {
	if err == nil {
		return nil
	}

	if apiErr, ok := err.(*core.Error); ok {
		return apiErr
	}

	if c.customConverter != nil {
		return c.customConverter(err)
	}

	return core.ConvertFrameworkError(err)
}
