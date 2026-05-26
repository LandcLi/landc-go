package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// ConsoleLogger 默认的控制台日志适配器
type ConsoleLogger struct {
	level          atomic.Int32
	fields         []Field
	ctx            context.Context
	logger         *log.Logger
	name           string
	format         string
	maxMessageSize int
}

// NewConsoleLogger 创建控制台日志实例
func NewConsoleLogger(name string, opts ...Option) *ConsoleLogger {
	options := DefaultLoggerOptions()
	for _, opt := range opts {
		opt(options)
	}

	var output interface {
		Write(p []byte) (n int, err error)
	}
	if options.OutputPath == "stdout" {
		output = os.Stdout
	} else {
		output = &lumberjack.Logger{
			Filename:   options.OutputPath,
			MaxSize:    int(options.MaxLogSize),
			MaxAge:     int(options.MaxLogAge.Hours() / 24),
			MaxBackups: options.MaxLogFiles,
			Compress:   options.CompressLogs,
		}
	}

	cl := &ConsoleLogger{
		fields:         make([]Field, 0),
		ctx:            context.Background(),
		logger:         log.New(output, "", 0),
		name:           name,
		format:         options.Format,
		maxMessageSize: options.MaxMessageSize,
	}
	cl.level.Store(int32(options.Level))
	return cl
}

// SetLevel 设置日志级别（并发安全）
func (c *ConsoleLogger) SetLevel(level LogLevel) {
	c.level.Store(int32(level))
}

// GetLevel 获取当前日志级别（并发安全）
func (c *ConsoleLogger) GetLevel() LogLevel {
	return LogLevel(c.level.Load())
}

// formatMessage 格式化日志消息
func (c *ConsoleLogger) formatMessage(level LogLevel, msg string, fields []Field) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	allFields := MergeFields(c.fields, fields)

	if c.format == "json" {
		jsonFields := make(map[string]interface{})
		jsonFields["time"] = timestamp
		jsonFields["level"] = level.String()
		jsonFields["logger"] = c.name
		jsonFields["msg"] = msg

		for _, field := range allFields {
			jsonFields[field.Key] = field.Value
		}

		jsonBytes, err := json.Marshal(jsonFields)
		if err != nil {
			fieldStr := ""
			for _, field := range allFields {
				fieldStr += fmt.Sprintf(" %s=%v", field.Key, field.Value)
			}
			formattedMsg := fmt.Sprintf("%s [%s] [%s] %s%s", timestamp, level.String(), c.name, msg, fieldStr)
			return SafeLimitMessageSize(formattedMsg, c.maxMessageSize)
		}

		return SafeLimitMessageSize(string(jsonBytes), c.maxMessageSize)
	}

	fieldStr := ""
	for _, field := range allFields {
		fieldStr += fmt.Sprintf(" %s=%v", field.Key, field.Value)
	}
	formattedMsg := fmt.Sprintf("%s [%s] [%s] %s%s", timestamp, level.String(), c.name, msg, fieldStr)
	return SafeLimitMessageSize(formattedMsg, c.maxMessageSize)
}

// Debug 输出调试级日志
func (c *ConsoleLogger) Debug(msg string, fields ...Field) {
	if c.GetLevel() <= DebugLevel {
		c.logger.Println(c.formatMessage(DebugLevel, msg, fields))
	}
}

// Debugf 输出格式化的调试级日志
func (c *ConsoleLogger) Debugf(format string, args ...interface{}) {
	if c.GetLevel() <= DebugLevel {
		msg := fmt.Sprintf(format, args...)
		c.logger.Println(c.formatMessage(DebugLevel, msg, nil))
	}
}

// Info 输出信息级日志
func (c *ConsoleLogger) Info(msg string, fields ...Field) {
	if c.GetLevel() <= InfoLevel {
		c.logger.Println(c.formatMessage(InfoLevel, msg, fields))
	}
}

// Infof 输出格式化的信息级日志
func (c *ConsoleLogger) Infof(format string, args ...interface{}) {
	if c.GetLevel() <= InfoLevel {
		msg := fmt.Sprintf(format, args...)
		c.logger.Println(c.formatMessage(InfoLevel, msg, nil))
	}
}

// Warn 输出警告级日志
func (c *ConsoleLogger) Warn(msg string, fields ...Field) {
	if c.GetLevel() <= WarnLevel {
		c.logger.Println(c.formatMessage(WarnLevel, msg, fields))
	}
}

// Warnf 输出格式化的警告级日志
func (c *ConsoleLogger) Warnf(format string, args ...interface{}) {
	if c.GetLevel() <= WarnLevel {
		msg := fmt.Sprintf(format, args...)
		c.logger.Println(c.formatMessage(WarnLevel, msg, nil))
	}
}

// Error 输出错误级日志
func (c *ConsoleLogger) Error(msg string, fields ...Field) {
	if c.GetLevel() <= ErrorLevel {
		c.logger.Println(c.formatMessage(ErrorLevel, msg, fields))
	}
}

// Errorf 输出格式化的错误级日志
func (c *ConsoleLogger) Errorf(format string, args ...interface{}) {
	if c.GetLevel() <= ErrorLevel {
		msg := fmt.Sprintf(format, args...)
		c.logger.Println(c.formatMessage(ErrorLevel, msg, nil))
	}
}

// Fatal 输出致命级日志并退出程序
func (c *ConsoleLogger) Fatal(msg string, fields ...Field) {
	if c.GetLevel() <= FatalLevel {
		c.logger.Println(c.formatMessage(FatalLevel, msg, fields))
		os.Exit(1)
	}
}

// Fatalf 输出格式化的致命级日志并退出程序
func (c *ConsoleLogger) Fatalf(format string, args ...interface{}) {
	if c.GetLevel() <= FatalLevel {
		msg := fmt.Sprintf(format, args...)
		c.logger.Println(c.formatMessage(FatalLevel, msg, nil))
		os.Exit(1)
	}
}

// Panic 输出恐慌级日志并触发panic
func (c *ConsoleLogger) Panic(msg string, fields ...Field) {
	if c.GetLevel() <= PanicLevel {
		formatted := c.formatMessage(PanicLevel, msg, fields)
		c.logger.Println(formatted)
		panic(formatted)
	}
}

// Panicf 输出格式化的恐慌级日志并触发panic
func (c *ConsoleLogger) Panicf(format string, args ...interface{}) {
	if c.GetLevel() <= PanicLevel {
		msg := fmt.Sprintf(format, args...)
		fullMsg := c.formatMessage(PanicLevel, msg, nil)
		c.logger.Println(fullMsg)
		panic(fullMsg)
	}
}

// WithFields 添加字段到日志（安全复制，避免 slice 共享）
func (c *ConsoleLogger) WithFields(fields ...Field) Logger {
	newLogger := &ConsoleLogger{
		fields:         MergeFields(c.fields, fields),
		ctx:            c.ctx,
		logger:         c.logger,
		name:           c.name,
		format:         c.format,
		maxMessageSize: c.maxMessageSize,
	}
	newLogger.level.Store(c.level.Load())
	return newLogger
}

// WithField 添加单个字段到日志
func (c *ConsoleLogger) WithField(key string, value interface{}) Logger {
	return c.WithFields(Field{Key: key, Value: value})
}

// WithContext 添加上下文到日志
func (c *ConsoleLogger) WithContext(ctx context.Context) Logger {
	newLogger := &ConsoleLogger{
		fields:         CopyFields(c.fields),
		ctx:            ctx,
		logger:         c.logger,
		name:           c.name,
		format:         c.format,
		maxMessageSize: c.maxMessageSize,
	}
	newLogger.level.Store(c.level.Load())
	return newLogger
}

// WithError 添加错误信息到日志
func (c *ConsoleLogger) WithError(err error) Logger {
	return c.WithField("error", err)
}

// WithTime 添加时间到日志
func (c *ConsoleLogger) WithTime(t time.Time) Logger {
	return c.WithField("time", t)
}

// IsDebugEnabled 检查调试级别是否启用
func (c *ConsoleLogger) IsDebugEnabled() bool {
	return c.GetLevel() <= DebugLevel
}

// IsInfoEnabled 检查信息级别是否启用
func (c *ConsoleLogger) IsInfoEnabled() bool {
	return c.GetLevel() <= InfoLevel
}

// IsWarnEnabled 检查警告级别是否启用
func (c *ConsoleLogger) IsWarnEnabled() bool {
	return c.GetLevel() <= WarnLevel
}

// IsErrorEnabled 检查错误级别是否启用
func (c *ConsoleLogger) IsErrorEnabled() bool {
	return c.GetLevel() <= ErrorLevel
}

// IsFatalEnabled 检查致命级别是否启用
func (c *ConsoleLogger) IsFatalEnabled() bool {
	return c.GetLevel() <= FatalLevel
}

// IsPanicEnabled 检查恐慌级别是否启用
func (c *ConsoleLogger) IsPanicEnabled() bool {
	return c.GetLevel() <= PanicLevel
}

// Sync 刷新日志缓冲区
func (c *ConsoleLogger) Sync() error {
	return nil
}

// ConsoleLoggerProvider 控制台日志提供者
type ConsoleLoggerProvider struct{}

// NewConsoleLoggerProvider 创建控制台日志提供者
func NewConsoleLoggerProvider() *ConsoleLoggerProvider {
	return &ConsoleLoggerProvider{}
}

// Create 创建日志实例
func (p *ConsoleLoggerProvider) Create(name string, opts ...Option) Logger {
	return NewConsoleLogger(name, opts...)
}

// CreateWithConfig 根据配置创建日志实例
func (p *ConsoleLoggerProvider) CreateWithConfig(name string, config map[string]interface{}) Logger {
	return NewConsoleLogger(name, parseConfigToOptions(config)...)
}

// parseConfigToOptions 将 map 配置转换为 Option 列表（公共辅助函数，减少代码重复）
func parseConfigToOptions(config map[string]interface{}) []Option {
	var opts []Option

	if lvl, ok := config["level"].(LogLevel); ok {
		opts = append(opts, WithLevel(lvl))
	}
	if format, ok := config["format"].(string); ok {
		opts = append(opts, WithFormat(format))
	}
	if path, ok := config["outputPath"].(string); ok {
		opts = append(opts, WithOutputPath(path))
	}
	if size, ok := config["maxLogSize"].(int64); ok {
		opts = append(opts, WithMaxLogSize(size))
	}
	if age, ok := config["maxLogAge"].(time.Duration); ok {
		opts = append(opts, WithMaxLogAge(age))
	}
	if files, ok := config["maxLogFiles"].(int); ok {
		opts = append(opts, WithMaxLogFiles(files))
	}
	if compress, ok := config["compressLogs"].(bool); ok {
		opts = append(opts, WithCompressLogs(compress))
	}
	if size, ok := config["maxMessageSize"].(int); ok {
		opts = append(opts, WithMaxMessageSize(size))
	}

	return opts
}
