package tests

import (
	"context"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/log/facade"
)

// CustomLogger 自定义日志实现
type CustomLogger struct {
	name string
}

// NewCustomLogger 创建自定义日志实例
func NewCustomLogger(name string) *CustomLogger {
	return &CustomLogger{name: name}
}

// Debug 输出调试级日志
func (c *CustomLogger) Debug(msg string, fields ...facade.Field) {}

// Debugf 输出格式化的调试级日志
func (c *CustomLogger) Debugf(format string, args ...interface{}) {}

// Info 输出信息级日志
func (c *CustomLogger) Info(msg string, fields ...facade.Field) {}

// Infof 输出格式化的信息级日志
func (c *CustomLogger) Infof(format string, args ...interface{}) {}

// Warn 输出警告级日志
func (c *CustomLogger) Warn(msg string, fields ...facade.Field) {}

// Warnf 输出格式化的警告级日志
func (c *CustomLogger) Warnf(format string, args ...interface{}) {}

// Error 输出错误级日志
func (c *CustomLogger) Error(msg string, fields ...facade.Field) {}

// Errorf 输出格式化的错误级日志
func (c *CustomLogger) Errorf(format string, args ...interface{}) {}

// Fatal 输出致命级日志并退出程序
func (c *CustomLogger) Fatal(msg string, fields ...facade.Field) {}

// Fatalf 输出格式化的致命级日志并退出程序
func (c *CustomLogger) Fatalf(format string, args ...interface{}) {}

// Panic 输出恐慌级日志并触发panic
func (c *CustomLogger) Panic(msg string, fields ...facade.Field) {}

// Panicf 输出格式化的恐慌级日志并触发panic
func (c *CustomLogger) Panicf(format string, args ...interface{}) {}

// WithFields 添加字段到日志
func (c *CustomLogger) WithFields(fields ...facade.Field) facade.Logger {
	return c
}

// WithField 添加单个字段到日志
func (c *CustomLogger) WithField(key string, value interface{}) facade.Logger {
	return c
}

// WithContext 添加上下文到日志
func (c *CustomLogger) WithContext(ctx context.Context) facade.Logger {
	return c
}

// WithError 添加错误信息到日志
func (c *CustomLogger) WithError(err error) facade.Logger {
	return c
}

// WithTime 添加时间到日志
func (c *CustomLogger) WithTime(t time.Time) facade.Logger {
	return c
}

// SetLevel 设置日志级别
func (c *CustomLogger) SetLevel(level facade.LogLevel) {}

// GetLevel 获取日志级别
func (c *CustomLogger) GetLevel() facade.LogLevel {
	return facade.InfoLevel
}

// IsDebugEnabled 检查调试级别是否启用
func (c *CustomLogger) IsDebugEnabled() bool {
	return false
}

// IsInfoEnabled 检查信息级别是否启用
func (c *CustomLogger) IsInfoEnabled() bool {
	return true
}

// IsWarnEnabled 检查警告级别是否启用
func (c *CustomLogger) IsWarnEnabled() bool {
	return true
}

// IsErrorEnabled 检查错误级别是否启用
func (c *CustomLogger) IsErrorEnabled() bool {
	return true
}

// IsFatalEnabled 检查致命级别是否启用
func (c *CustomLogger) IsFatalEnabled() bool {
	return true
}

// IsPanicEnabled 检查恐慌级别是否启用
func (c *CustomLogger) IsPanicEnabled() bool {
	return true
}

// Sync 同步日志
func (c *CustomLogger) Sync() error {
	return nil
}

// CustomLoggerProvider 自定义日志提供者
type CustomLoggerProvider struct{}

// Create 创建日志实例
func (p *CustomLoggerProvider) Create(name string, opts ...facade.Option) facade.Logger {
	return NewCustomLogger(name)
}

// CreateWithConfig 根据配置创建日志实例
func (p *CustomLoggerProvider) CreateWithConfig(name string, config map[string]interface{}) facade.Logger {
	return NewCustomLogger(name)
}

func TestCustomLoggerProvider(t *testing.T) {
	// 注册自定义提供者
	facade.RegisterProvider("custom", &CustomLoggerProvider{})

	// 使用自定义提供者
	customLogger := facade.GetLoggerWithProvider("app", "custom")
	if customLogger == nil {
		t.Fatal("自定义日志提供者创建失败")
	}

	customLogger.Info("使用自定义日志提供者")

	// 注销自定义提供者
	facade.UnregisterProvider("custom")
}
