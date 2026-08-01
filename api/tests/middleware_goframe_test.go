//go:build goframe
// +build goframe

package tests

import (
	"errors"
	"testing"

	"github.com/LandcLi/landc-go/api/core"
	"github.com/LandcLi/landc-go/api/middleware/goframe"
)

// TestGoFrameErrorConverter 验证 GoFrame 错误转换器（仅 `-tags goframe` 时编译）
func TestGoFrameErrorConverter(t *testing.T) {
	converter := goframe.NewGoFrameErrorConverter()

	apiErr := core.NewError(core.ErrorCodeBadRequest, "Bad Request")
	convertedErr := converter.ConvertError(apiErr)
	if convertedErr != apiErr {
		t.Error("ConvertError should return the same core.Error instance")
	}

	stdErr := errors.New("GoFrame error")
	convertedStdErr := converter.ConvertGoFrameError(stdErr)
	if convertedStdErr == nil {
		t.Fatal("ConvertGoFrameError should not return nil")
	}
	if convertedStdErr.Code != core.ErrorCodeInternalServerError {
		t.Errorf("Expected code %d, got %d", core.ErrorCodeInternalServerError, convertedStdErr.Code)
	}

	// 自定义转换器
	customConverter := goframe.NewGoFrameErrorConverter().WithCustomConverter(func(err error) *core.Error {
		return core.NewError(core.ErrorCodeExternalAPIError, "Custom External API Error")
	})
	customErr := customConverter.ConvertError(stdErr)
	if customErr.Code != core.ErrorCodeExternalAPIError {
		t.Errorf("Expected code %d, got %d", core.ErrorCodeExternalAPIError, customErr.Code)
	}

	// nil error
	if converter.ConvertGoFrameError(nil) != nil {
		t.Error("ConvertGoFrameError(nil) should return nil")
	}
}
