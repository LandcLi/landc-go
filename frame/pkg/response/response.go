package response

import (
	"net/http"
	"time"

	"github.com/LandcLi/landc-go/api/core"
	"github.com/gin-gonic/gin"
)

// 直接复用 landc-go/api 的错误码体系
const (
	CodeSuccess             = int(core.CodeSuccess)
	CodeBadRequest          = int(core.ErrorCodeBadRequest)
	CodeInvalidParams       = int(core.ErrorCodeInvalidParams)
	CodeBusinessError       = int(core.ErrorCodeBusinessError)
	CodeUnauthorized        = int(core.ErrorCodeUnauthorized)
	CodeForbidden           = int(core.ErrorCodeForbidden)
	CodeNotFound            = int(core.ErrorCodeNotFound)
	CodeValidationError     = int(core.ErrorCodeValidationFailed)
	CodeInternalServerError = int(core.ErrorCodeInternalServerError)
	CodeDatabaseError       = int(core.ErrorCodeDatabaseError)
	CodeExternalAPIError    = int(core.ErrorCodeExternalAPIError)
)

// Response 使用 landc-go/api 的标准响应结构
type Response = core.Response

// PageData 分页数据
type PageData struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, &core.Response{
		Code:      int(core.CodeSuccess),
		Message:   "success",
		Data:      data,
		TraceID:   c.GetString("trace_id"),
		Latency:   getLatency(c),
		Timestamp: time.Now(),
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, &core.Response{
		Code:      int(core.CodeSuccess),
		Message:   message,
		Data:      data,
		TraceID:   c.GetString("trace_id"),
		Latency:   getLatency(c),
		Timestamp: time.Now(),
	})
}

// SuccessPage 分页成功响应
func SuccessPage(c *gin.Context, list interface{}, total int64, page, size int) {
	Success(c, PageData{
		List:  list,
		Total: total,
		Page:  page,
		Size:  size,
	})
}

// Error 错误响应（HTTP 200，业务错误码在 body 中）
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, &core.Response{
		Code:      code,
		Message:   message,
		TraceID:   c.GetString("trace_id"),
		Latency:   getLatency(c),
		Timestamp: time.Now(),
	})
}

// ErrorWithDetails 带详情的错误响应
func ErrorWithDetails(c *gin.Context, code int, message string, details interface{}) {
	c.JSON(http.StatusOK, &core.Response{
		Code:      code,
		Message:   message,
		Details:   details,
		TraceID:   c.GetString("trace_id"),
		Latency:   getLatency(c),
		Timestamp: time.Now(),
	})
}

// ErrorWithHTTPStatus 带 HTTP 状态码的错误响应
func ErrorWithHTTPStatus(c *gin.Context, httpStatus, code int, message string) {
	c.JSON(httpStatus, &core.Response{
		Code:      code,
		Message:   message,
		TraceID:   c.GetString("trace_id"),
		Latency:   getLatency(c),
		Timestamp: time.Now(),
	})
}

// ErrorFromCoreError 从 landc-go/api 的 Error 对象构建响应
func ErrorFromCoreError(c *gin.Context, err *core.Error) {
	if err == nil {
		Success(c, nil)
		return
	}
	Error(c, int(err.Code), err.Message)
}

// BadRequest 400 错误
func BadRequest(c *gin.Context, message string) {
	Error(c, CodeBadRequest, message)
}

// Unauthorized 401 错误
func Unauthorized(c *gin.Context, message string) {
	ErrorWithHTTPStatus(c, http.StatusUnauthorized, CodeUnauthorized, message)
}

// Forbidden 403 错误
func Forbidden(c *gin.Context, message string) {
	ErrorWithHTTPStatus(c, http.StatusForbidden, CodeForbidden, message)
}

// NotFound 404 错误
func NotFound(c *gin.Context, message string) {
	ErrorWithHTTPStatus(c, http.StatusNotFound, CodeNotFound, message)
}

// InternalServerError 500 错误
func InternalServerError(c *gin.Context, message string) {
	Error(c, CodeInternalServerError, message)
}

// DatabaseError 数据库错误
func DatabaseError(c *gin.Context, message string) {
	Error(c, CodeDatabaseError, message)
}

// ValidationError 验证错误
func ValidationError(c *gin.Context, message string) {
	Error(c, CodeValidationError, message)
}

func getLatency(c *gin.Context) int64 {
	if startTime, ok := c.Get("_start_time"); ok {
		if t, ok := startTime.(time.Time); ok {
			return time.Since(t).Milliseconds()
		}
	}
	return 0
}
