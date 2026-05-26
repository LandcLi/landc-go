package gin

import (
	"errors"

	"github.com/LandcLi/landc-go/api/core"
	"github.com/gin-gonic/gin"
)

// 编译时接口验证
var _ core.ErrorConverter = (*GinErrorConverter)(nil)

// GinErrorConverter Gin框架错误转换器
type GinErrorConverter struct {
	customConverter func(err error) *core.Error
}

// NewGinErrorConverter 创建Gin错误转换器
func NewGinErrorConverter() *GinErrorConverter {
	return &GinErrorConverter{}
}

// WithCustomConverter 设置自定义转换器
func (c *GinErrorConverter) WithCustomConverter(converter func(err error) *core.Error) *GinErrorConverter {
	c.customConverter = converter
	return c
}

// ConvertError 实现 ErrorConverter 接口
func (c *GinErrorConverter) ConvertError(err error) *core.Error {
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
func (c *GinErrorConverter) ConvertResponse(data interface{}) *core.Response {
	converter := core.NewDefaultErrorConverter()
	return converter.ConvertResponse(data)
}

// ConvertGinError 转换 Gin 框架特有的错误
func (c *GinErrorConverter) ConvertGinError(err error) *core.Error {
	if err == nil {
		return nil
	}

	if apiErr, ok := err.(*core.Error); ok {
		return apiErr
	}

	if c.customConverter != nil {
		return c.customConverter(err)
	}

	var ginErr *gin.Error
	if errors.As(err, &ginErr) {
		// 优先检查内部错误是否为 core.Error
		if apiErr, ok := ginErr.Err.(*core.Error); ok {
			return apiErr
		}

		switch ginErr.Type {
		case gin.ErrorTypeBind:
			return core.NewError(core.ErrorCodeValidationFailed, "Validation Failed")
		case gin.ErrorTypeRender:
			return core.NewError(core.ErrorCodeInternalServerError, "Render Error")
		case gin.ErrorTypePublic:
			return core.NewError(core.ErrorCodeBadRequest, ginErr.Error())
		case gin.ErrorTypePrivate:
			return core.NewErrorWithCause(core.ErrorCodeInternalServerError, "Internal Server Error", ginErr.Err)
		default:
			return core.NewErrorWithCause(core.ErrorCodeInternalServerError, "Internal Server Error", ginErr.Err)
		}
	}

	return core.ConvertFrameworkError(err)
}
