package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/api/core"
)

func TestErrorCreation(t *testing.T) {
	err := core.NewError(core.ErrorCodeBadRequest, "Bad Request")
	if err == nil {
		t.Fatal("NewError should not return nil")
	}
	if err.Code != core.ErrorCodeBadRequest {
		t.Errorf("Expected error code %d, got %d", core.ErrorCodeBadRequest, err.Code)
	}
	if err.Message != "Bad Request" {
		t.Errorf("Expected error message 'Bad Request', got '%s'", err.Message)
	}

	details := map[string]string{"field": "email", "message": "Invalid email format"}
	errWithDetails := core.NewErrorWithDetails(core.ErrorCodeValidationFailed, "Validation Failed", details)
	if errWithDetails == nil {
		t.Fatal("NewErrorWithDetails should not return nil")
	}
	if errWithDetails.Details == nil {
		t.Errorf("Expected error details, got nil")
	}

	cause := errors.New("database connection error")
	errWithCause := core.NewErrorWithCause(core.ErrorCodeDatabaseError, "Database Error", cause)
	if errWithCause == nil {
		t.Fatal("NewErrorWithCause should not return nil")
	}
	if errWithCause.Cause != cause {
		t.Errorf("Expected error cause %v, got %v", cause, errWithCause.Cause)
	}
}

func TestNewCustomErrorValidation(t *testing.T) {
	// 有效范围
	customErr, err := core.NewCustomError(60000, "Custom Error")
	if err != nil {
		t.Fatalf("NewCustomError(60000) should not return error: %v", err)
	}
	if customErr.Code != core.ErrorCode(60000) {
		t.Errorf("Expected error code 60000, got %d", customErr.Code)
	}

	customErr2, err := core.NewCustomError(99999, "Max Custom")
	if err != nil {
		t.Fatalf("NewCustomError(99999) should not return error: %v", err)
	}
	if customErr2.Code != core.ErrorCode(99999) {
		t.Errorf("Expected error code 99999, got %d", customErr2.Code)
	}

	// 无效范围 - 过小
	_, err = core.NewCustomError(59999, "Too Small")
	if err == nil {
		t.Error("NewCustomError(59999) should return error")
	}

	// 无效范围 - 过大
	_, err = core.NewCustomError(100000, "Too Large")
	if err == nil {
		t.Error("NewCustomError(100000) should return error")
	}

	// 无效范围 - 使用标准错误码
	_, err = core.NewCustomError(40000, "Standard Code")
	if err == nil {
		t.Error("NewCustomError(40000) should return error")
	}
}

func TestNewCustomErrorWithDetails(t *testing.T) {
	details := map[string]string{"key": "value"}
	customErr, err := core.NewCustomErrorWithDetails(70000, "Custom", details)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if customErr.Details == nil {
		t.Error("Expected details, got nil")
	}

	_, err = core.NewCustomErrorWithDetails(10000, "Invalid", details)
	if err == nil {
		t.Error("Should reject code out of custom range")
	}
}

func TestNewCustomErrorWithCause(t *testing.T) {
	cause := errors.New("root cause")
	customErr, err := core.NewCustomErrorWithCause(80000, "Custom", cause)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if customErr.Cause != cause {
		t.Error("Expected cause to be set")
	}

	_, err = core.NewCustomErrorWithCause(10000, "Invalid", cause)
	if err == nil {
		t.Error("Should reject code out of custom range")
	}
}

func TestErrorMethods(t *testing.T) {
	err := core.NewError(core.ErrorCodeInternalServerError, "Internal Server Error")
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Error() should return non-empty string")
	}
	if errMsg != "[50000] Internal Server Error" {
		t.Errorf("Unexpected error string: %s", errMsg)
	}

	cause := errors.New("underlying error")
	errWithCause := core.NewErrorWithCause(core.ErrorCodeBadRequest, "Bad Request", cause)
	errMsgWithCause := errWithCause.Error()
	if errMsgWithCause != "[40000] Bad Request: underlying error" {
		t.Errorf("Unexpected error with cause string: %s", errMsgWithCause)
	}

	if errWithCause.Unwrap() != cause {
		t.Error("Unwrap() should return the cause error")
	}
}

func TestErrorClone(t *testing.T) {
	original := core.NewError(core.ErrorCodeBadRequest, "Bad Request")
	cloned := original.Clone()

	if cloned == original {
		t.Error("Clone should return a different pointer")
	}
	if cloned.Code != original.Code || cloned.Message != original.Message {
		t.Error("Clone should have the same Code and Message")
	}

	// 修改克隆不应影响原始
	cloned.Message = "Modified"
	if original.Message == "Modified" {
		t.Error("Modifying clone should not affect original")
	}
}

func TestPredefinedErrors(t *testing.T) {
	// 每次调用返回新实例
	err1 := core.ErrBadRequest()
	err2 := core.ErrBadRequest()

	if err1 == err2 {
		t.Error("Predefined error functions should return different instances")
	}
	if err1.Code != core.ErrorCodeBadRequest {
		t.Errorf("Expected code %d, got %d", core.ErrorCodeBadRequest, err1.Code)
	}

	// 修改一个不影响另一个
	err1.Message = "Modified"
	if err2.Message == "Modified" {
		t.Error("Modifying one predefined error should not affect others")
	}
}

func TestIsCustomError(t *testing.T) {
	if core.IsCustomError(core.ErrorCodeBadRequest) {
		t.Error("ErrorCodeBadRequest should not be a custom error")
	}
	if !core.IsCustomError(core.ErrorCode(60000)) {
		t.Error("ErrorCode 60000 should be a custom error")
	}
	if !core.IsCustomError(core.ErrorCode(99999)) {
		t.Error("ErrorCode 99999 should be a custom error")
	}
	if core.IsCustomError(core.ErrorCode(59999)) {
		t.Error("ErrorCode 59999 should not be a custom error")
	}
	if core.IsCustomError(core.ErrorCode(100000)) {
		t.Error("ErrorCode 100000 should not be a custom error")
	}
}

func TestConvertFrameworkError(t *testing.T) {
	// nil error
	if core.ConvertFrameworkError(nil) != nil {
		t.Error("ConvertFrameworkError(nil) should return nil")
	}

	// core.Error 直接返回
	apiErr := core.NewError(core.ErrorCodeBadRequest, "Bad Request")
	converted := core.ConvertFrameworkError(apiErr)
	if converted != apiErr {
		t.Error("ConvertFrameworkError should return the same core.Error")
	}

	// 普通 error 转为 InternalServerError
	stdErr := errors.New("some error")
	converted = core.ConvertFrameworkError(stdErr)
	if converted.Code != core.ErrorCodeInternalServerError {
		t.Errorf("Expected code %d, got %d", core.ErrorCodeInternalServerError, converted.Code)
	}
	if converted.Cause != stdErr {
		t.Error("Expected cause to be the original error")
	}
}

func TestResponseBuilder(t *testing.T) {
	builder := core.NewResponseBuilder()

	data := map[string]string{"message": "Hello, World!"}
	resp := builder.BuildSuccessResponse(data, "test-trace-id", 100*time.Millisecond)
	if resp == nil {
		t.Fatal("BuildSuccessResponse should not return nil")
	}
	if resp.Code != int(core.CodeSuccess) {
		t.Errorf("Expected response code %d, got %d", core.CodeSuccess, resp.Code)
	}
	if resp.Message != "success" {
		t.Errorf("Expected message 'success', got '%s'", resp.Message)
	}
	if resp.Data == nil {
		t.Error("Expected response data, got nil")
	}
	if resp.TraceID != "test-trace-id" {
		t.Errorf("Expected trace ID 'test-trace-id', got '%s'", resp.TraceID)
	}
	if resp.Latency != 100 {
		t.Errorf("Expected latency 100 (ms), got %d", resp.Latency)
	}

	err := core.NewError(core.ErrorCodeBadRequest, "Bad Request")
	errResp := builder.BuildErrorResponse(err, "test-trace-id", 50*time.Millisecond)
	if errResp == nil {
		t.Fatal("BuildErrorResponse should not return nil")
	}
	if errResp.Code != int(core.ErrorCodeBadRequest) {
		t.Errorf("Expected error code %d, got %d", core.ErrorCodeBadRequest, errResp.Code)
	}
	if errResp.Latency != 50 {
		t.Errorf("Expected latency 50 (ms), got %d", errResp.Latency)
	}

	// 自定义 ResponseBuilder
	customBuilder := core.NewResponseBuilder().WithResponseBuilder(func(data interface{}) *core.Response {
		return &core.Response{
			Code:      20000,
			Message:   "Custom Success",
			Data:      data,
			Timestamp: time.Now(),
		}
	})
	customResp := customBuilder.BuildSuccessResponse(data, "test-trace-id", 200*time.Millisecond)
	if customResp.Code != 20000 {
		t.Errorf("Expected custom response code 20000, got %d", customResp.Code)
	}
	if customResp.Message != "Custom Success" {
		t.Errorf("Expected 'Custom Success', got '%s'", customResp.Message)
	}
}

func TestDefaultErrorConverter(t *testing.T) {
	converter := core.NewDefaultErrorConverter()

	apiErr := core.NewError(core.ErrorCodeBadRequest, "Bad Request")
	convertedErr := converter.ConvertError(apiErr)
	if convertedErr != apiErr {
		t.Error("ConvertError should return the same core.Error instance")
	}

	stdErr := errors.New("standard error")
	convertedStdErr := converter.ConvertError(stdErr)
	if convertedStdErr == nil {
		t.Fatal("ConvertError should not return nil for standard error")
	}
	if convertedStdErr.Code != core.ErrorCodeInternalServerError {
		t.Errorf("Expected code %d, got %d", core.ErrorCodeInternalServerError, convertedStdErr.Code)
	}

	nilErr := converter.ConvertError(nil)
	if nilErr != nil {
		t.Error("ConvertError should return nil for nil error")
	}

	response := &core.Response{Code: int(core.CodeSuccess), Message: "Success", Data: "data"}
	convertedResp := converter.ConvertResponse(response)
	if convertedResp != response {
		t.Error("ConvertResponse should return the same Response instance")
	}

	data := map[string]string{"key": "value"}
	convertedData := converter.ConvertResponse(data)
	if convertedData == nil {
		t.Fatal("ConvertResponse should not return nil for data")
	}
	if convertedData.Code != int(core.CodeSuccess) {
		t.Errorf("Expected code %d, got %d", core.CodeSuccess, convertedData.Code)
	}
	if convertedData.Message != "success" {
		t.Errorf("Expected message 'success', got '%s'", convertedData.Message)
	}
}

func TestResponseSetLatency(t *testing.T) {
	resp := &core.Response{}
	resp.SetLatency(150 * time.Millisecond)
	if resp.Latency != 150 {
		t.Errorf("Expected latency 150, got %d", resp.Latency)
	}

	resp.SetLatency(2 * time.Second)
	if resp.Latency != 2000 {
		t.Errorf("Expected latency 2000, got %d", resp.Latency)
	}
}
