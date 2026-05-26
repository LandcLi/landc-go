package core

import "fmt"

// ErrorCode 统一的错误码类型
type ErrorCode int

const (
	// 成功
	CodeSuccess ErrorCode = 10000

	// 客户端错误 4xxxx
	ErrorCodeBadRequest          ErrorCode = 40000
	ErrorCodeInvalidParams       ErrorCode = 40001
	ErrorCodeBusinessError       ErrorCode = 40002
	ErrorCodeUnauthorized        ErrorCode = 40100
	ErrorCodeForbidden           ErrorCode = 40300
	ErrorCodeNotFound            ErrorCode = 40400
	ErrorCodeMethodNotAllowed    ErrorCode = 40500
	ErrorCodeConflict            ErrorCode = 40900
	ErrorCodeUnprocessableEntity ErrorCode = 42200
	ErrorCodeValidationFailed    ErrorCode = 42201

	// 服务端错误 5xxxx
	CodeError                       ErrorCode = 50000
	ErrorCodeInternalServerError    ErrorCode = 50000
	ErrorCodeDatabaseError          ErrorCode = 50001
	ErrorCodeExternalAPIError       ErrorCode = 50002
	ErrorCodeNotImplemented         ErrorCode = 50100
	ErrorCodeBadGateway             ErrorCode = 50200
	ErrorCodeServiceUnavailable     ErrorCode = 50300
	ErrorCodeGatewayTimeout         ErrorCode = 50400

	// 自定义错误码范围
	ErrorCodeCustomMin ErrorCode = 60000
	ErrorCodeCustomMax ErrorCode = 99999
)

// Error 标准化的错误结构
type Error struct {
	Code    ErrorCode
	Message string
	Details interface{}
	Cause   error
}

// NewError 创建错误
func NewError(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithDetails 创建带详情的错误
func NewErrorWithDetails(code ErrorCode, message string, details interface{}) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewErrorWithCause 创建带原因的错误
func NewErrorWithCause(code ErrorCode, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Error 实现 error 接口
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 支持 errors.Is / errors.As
func (e *Error) Unwrap() error {
	return e.Cause
}

// Clone 返回错误的副本，防止全局预定义错误被意外修改
func (e *Error) Clone() *Error {
	return &Error{
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
		Cause:   e.Cause,
	}
}

// IsCustomError 检查错误码是否在自定义范围内
func IsCustomError(code ErrorCode) bool {
	return code >= ErrorCodeCustomMin && code <= ErrorCodeCustomMax
}

// NewCustomError 创建自定义错误（校验 code 必须在 60000-99999 范围内）
func NewCustomError(code int, message string) (*Error, error) {
	if code < int(ErrorCodeCustomMin) || code > int(ErrorCodeCustomMax) {
		return nil, fmt.Errorf("custom error code must be in range [%d, %d], got %d",
			ErrorCodeCustomMin, ErrorCodeCustomMax, code)
	}
	return &Error{
		Code:    ErrorCode(code),
		Message: message,
	}, nil
}

// NewCustomErrorWithDetails 创建带详情的自定义错误
func NewCustomErrorWithDetails(code int, message string, details interface{}) (*Error, error) {
	if code < int(ErrorCodeCustomMin) || code > int(ErrorCodeCustomMax) {
		return nil, fmt.Errorf("custom error code must be in range [%d, %d], got %d",
			ErrorCodeCustomMin, ErrorCodeCustomMax, code)
	}
	return &Error{
		Code:    ErrorCode(code),
		Message: message,
		Details: details,
	}, nil
}

// NewCustomErrorWithCause 创建带原因的自定义错误
func NewCustomErrorWithCause(code int, message string, cause error) (*Error, error) {
	if code < int(ErrorCodeCustomMin) || code > int(ErrorCodeCustomMax) {
		return nil, fmt.Errorf("custom error code must be in range [%d, %d], got %d",
			ErrorCodeCustomMin, ErrorCodeCustomMax, code)
	}
	return &Error{
		Code:    ErrorCode(code),
		Message: message,
		Cause:   cause,
	}, nil
}

// 预定义错误（使用函数返回副本，防止全局状态被意外修改）

func ErrBadRequest() *Error     { return NewError(ErrorCodeBadRequest, "Bad Request") }
func ErrUnauthorized() *Error   { return NewError(ErrorCodeUnauthorized, "Unauthorized") }
func ErrForbidden() *Error      { return NewError(ErrorCodeForbidden, "Forbidden") }
func ErrNotFound() *Error       { return NewError(ErrorCodeNotFound, "Not Found") }
func ErrMethodNotAllowed() *Error { return NewError(ErrorCodeMethodNotAllowed, "Method Not Allowed") }
func ErrConflict() *Error       { return NewError(ErrorCodeConflict, "Conflict") }
func ErrUnprocessableEntity() *Error { return NewError(ErrorCodeUnprocessableEntity, "Unprocessable Entity") }

func ErrInternalServerError() *Error { return NewError(ErrorCodeInternalServerError, "Internal Server Error") }
func ErrNotImplemented() *Error      { return NewError(ErrorCodeNotImplemented, "Not Implemented") }
func ErrBadGateway() *Error          { return NewError(ErrorCodeBadGateway, "Bad Gateway") }
func ErrServiceUnavailable() *Error  { return NewError(ErrorCodeServiceUnavailable, "Service Unavailable") }
func ErrGatewayTimeout() *Error      { return NewError(ErrorCodeGatewayTimeout, "Gateway Timeout") }

func ErrValidationFailed() *Error { return NewError(ErrorCodeValidationFailed, "Validation Failed") }
func ErrInvalidParams() *Error    { return NewError(ErrorCodeInvalidParams, "Invalid Parameters") }
func ErrDatabaseError() *Error    { return NewError(ErrorCodeDatabaseError, "Database Error") }
func ErrExternalAPIError() *Error { return NewError(ErrorCodeExternalAPIError, "External API Error") }
func ErrBusinessError() *Error    { return NewError(ErrorCodeBusinessError, "Business Error") }

// ConvertFrameworkError 公共的框架错误转换（供 Gin/GoFrame converter 复用）
func ConvertFrameworkError(err error) *Error {
	if err == nil {
		return nil
	}

	if apiErr, ok := err.(*Error); ok {
		return apiErr
	}

	return NewErrorWithCause(ErrorCodeInternalServerError, "Internal Server Error", err)
}
