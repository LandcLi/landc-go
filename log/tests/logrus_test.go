//go:build logrus_provider

package tests

import (
	"testing"

	"github.com/LandcLi/landc-go/log/facade"
	_ "github.com/LandcLi/landc-go/log/providers/logrus"
)

// TestLogrusProvider 测试Logrus提供者
func TestLogrusProvider(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test", "logrus",
		facade.WithLevel(facade.DebugLevel),
		facade.WithFormat("text"),
	)
	if logger == nil {
		t.Fatal("创建Logrus日志失败")
	}

	logger.Info("Logrus日志测试")
	logger.Debug("Logrus调试日志")
	logger.Warn("Logrus警告日志")
	logger.Error("Logrus错误日志")
}

// TestLogrusWithOptions 测试Logrus带选项
func TestLogrusWithOptions(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test", "logrus",
		facade.WithLevel(facade.InfoLevel),
		facade.WithFormat("json"),
		facade.WithOutputPath("stdout"),
		facade.WithMaxMessageSize(10),
	)
	if logger == nil {
		t.Fatal("创建Logrus日志失败")
	}

	logger.Info("Logrus带选项测试")
}
