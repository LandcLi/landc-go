//go:build !zap_provider && !logrus_provider && !gin_provider && !gf_provider

package tests

import (
	"testing"

	"github.com/LandcLi/landc-go/log/facade"
)

// TestCoreOnly 测试仅使用核心包（不引入任何第三方日志库）
func TestCoreOnly(t *testing.T) {
	logger := facade.GetLogger()
	if logger == nil {
		t.Fatal("获取全局日志失败")
	}

	logger.Info("核心包测试")
}

// TestConsoleProvider 测试控制台提供者
func TestConsoleProvider(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test", "console")
	if logger == nil {
		t.Fatal("创建控制台日志失败")
	}

	logger.Info("控制台日志测试")
}

// TestStdProvider 测试标准库提供者
func TestStdProvider(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test", "std")
	if logger == nil {
		t.Fatal("创建标准库日志失败")
	}

	logger.Info("标准库日志测试")
}
