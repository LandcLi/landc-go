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

// StdLogger 标准库log适配器
type StdLogger struct {
	level          atomic.Int32
	fields         []Field
	ctx            context.Context
	logger         *log.Logger
	name           string
	format         string
	maxMessageSize int
}

// NewStdLogger 创建标准库log实例
func NewStdLogger(name string, opts ...Option) *StdLogger {
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

	sl := &StdLogger{
		fields:         make([]Field, 0),
		ctx:            context.Background(),
		logger:         log.New(output, "", log.LstdFlags),
		name:           name,
		format:         options.Format,
		maxMessageSize: options.MaxMessageSize,
	}
	sl.level.Store(int32(options.Level))
	return sl
}

// SetLevel 设置日志级别（并发安全）
func (s *StdLogger) SetLevel(level LogLevel) {
	s.level.Store(int32(level))
}

// GetLevel 获取当前日志级别（并发安全）
func (s *StdLogger) GetLevel() LogLevel {
	return LogLevel(s.level.Load())
}

// formatMessage 格式化日志消息
func (s *StdLogger) formatMessage(level LogLevel, msg string, fields []Field) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	allFields := MergeFields(s.fields, fields)

	if s.format == "json" {
		jsonFields := make(map[string]interface{})
		jsonFields["time"] = timestamp
		jsonFields["level"] = level.String()
		jsonFields["logger"] = s.name
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
			formattedMsg := fmt.Sprintf("[%s] [%s] %s%s", level.String(), s.name, msg, fieldStr)
			return SafeLimitMessageSize(formattedMsg, s.maxMessageSize)
		}

		return SafeLimitMessageSize(string(jsonBytes), s.maxMessageSize)
	}

	fieldStr := ""
	for _, field := range allFields {
		fieldStr += fmt.Sprintf(" %s=%v", field.Key, field.Value)
	}
	formattedMsg := fmt.Sprintf("[%s] [%s] %s%s", level.String(), s.name, msg, fieldStr)
	return SafeLimitMessageSize(formattedMsg, s.maxMessageSize)
}

// Debug 输出调试级日志
func (s *StdLogger) Debug(msg string, fields ...Field) {
	if s.GetLevel() <= DebugLevel {
		s.logger.Println(s.formatMessage(DebugLevel, msg, fields))
	}
}

// Debugf 输出格式化的调试级日志
func (s *StdLogger) Debugf(format string, args ...interface{}) {
	if s.GetLevel() <= DebugLevel {
		msg := fmt.Sprintf(format, args...)
		s.logger.Println(s.formatMessage(DebugLevel, msg, nil))
	}
}

// Info 输出信息级日志
func (s *StdLogger) Info(msg string, fields ...Field) {
	if s.GetLevel() <= InfoLevel {
		s.logger.Println(s.formatMessage(InfoLevel, msg, fields))
	}
}

// Infof 输出格式化的信息级日志
func (s *StdLogger) Infof(format string, args ...interface{}) {
	if s.GetLevel() <= InfoLevel {
		msg := fmt.Sprintf(format, args...)
		s.logger.Println(s.formatMessage(InfoLevel, msg, nil))
	}
}

// Warn 输出警告级日志
func (s *StdLogger) Warn(msg string, fields ...Field) {
	if s.GetLevel() <= WarnLevel {
		s.logger.Println(s.formatMessage(WarnLevel, msg, fields))
	}
}

// Warnf 输出格式化的警告级日志
func (s *StdLogger) Warnf(format string, args ...interface{}) {
	if s.GetLevel() <= WarnLevel {
		msg := fmt.Sprintf(format, args...)
		s.logger.Println(s.formatMessage(WarnLevel, msg, nil))
	}
}

// Error 输出错误级日志
func (s *StdLogger) Error(msg string, fields ...Field) {
	if s.GetLevel() <= ErrorLevel {
		s.logger.Println(s.formatMessage(ErrorLevel, msg, fields))
	}
}

// Errorf 输出格式化的错误级日志
func (s *StdLogger) Errorf(format string, args ...interface{}) {
	if s.GetLevel() <= ErrorLevel {
		msg := fmt.Sprintf(format, args...)
		s.logger.Println(s.formatMessage(ErrorLevel, msg, nil))
	}
}

// Fatal 输出致命级日志并退出程序
func (s *StdLogger) Fatal(msg string, fields ...Field) {
	if s.GetLevel() <= FatalLevel {
		s.logger.Println(s.formatMessage(FatalLevel, msg, fields))
		os.Exit(1)
	}
}

// Fatalf 输出格式化的致命级日志并退出程序
func (s *StdLogger) Fatalf(format string, args ...interface{}) {
	if s.GetLevel() <= FatalLevel {
		msg := fmt.Sprintf(format, args...)
		s.logger.Println(s.formatMessage(FatalLevel, msg, nil))
		os.Exit(1)
	}
}

// Panic 输出恐慌级日志并触发panic
func (s *StdLogger) Panic(msg string, fields ...Field) {
	if s.GetLevel() <= PanicLevel {
		formatted := s.formatMessage(PanicLevel, msg, fields)
		s.logger.Println(formatted)
		panic(formatted)
	}
}

// Panicf 输出格式化的恐慌级日志并触发panic
func (s *StdLogger) Panicf(format string, args ...interface{}) {
	if s.GetLevel() <= PanicLevel {
		msg := fmt.Sprintf(format, args...)
		fullMsg := s.formatMessage(PanicLevel, msg, nil)
		s.logger.Println(fullMsg)
		panic(fullMsg)
	}
}

// WithFields 添加字段到日志（安全复制，避免 slice 共享）
func (s *StdLogger) WithFields(fields ...Field) Logger {
	newLogger := &StdLogger{
		fields:         MergeFields(s.fields, fields),
		ctx:            s.ctx,
		logger:         s.logger,
		name:           s.name,
		format:         s.format,
		maxMessageSize: s.maxMessageSize,
	}
	newLogger.level.Store(s.level.Load())
	return newLogger
}

// WithField 添加单个字段到日志
func (s *StdLogger) WithField(key string, value interface{}) Logger {
	return s.WithFields(Field{Key: key, Value: value})
}

// WithContext 添加上下文到日志
func (s *StdLogger) WithContext(ctx context.Context) Logger {
	newLogger := &StdLogger{
		fields:         CopyFields(s.fields),
		ctx:            ctx,
		logger:         s.logger,
		name:           s.name,
		format:         s.format,
		maxMessageSize: s.maxMessageSize,
	}
	newLogger.level.Store(s.level.Load())
	return newLogger
}

// WithError 添加错误信息到日志
func (s *StdLogger) WithError(err error) Logger {
	return s.WithField("error", err)
}

// WithTime 添加时间到日志
func (s *StdLogger) WithTime(t time.Time) Logger {
	return s.WithField("time", t)
}

// IsDebugEnabled 检查调试级别是否启用
func (s *StdLogger) IsDebugEnabled() bool {
	return s.GetLevel() <= DebugLevel
}

// IsInfoEnabled 检查信息级别是否启用
func (s *StdLogger) IsInfoEnabled() bool {
	return s.GetLevel() <= InfoLevel
}

// IsWarnEnabled 检查警告级别是否启用
func (s *StdLogger) IsWarnEnabled() bool {
	return s.GetLevel() <= WarnLevel
}

// IsErrorEnabled 检查错误级别是否启用
func (s *StdLogger) IsErrorEnabled() bool {
	return s.GetLevel() <= ErrorLevel
}

// IsFatalEnabled 检查致命级别是否启用
func (s *StdLogger) IsFatalEnabled() bool {
	return s.GetLevel() <= FatalLevel
}

// IsPanicEnabled 检查恐慌级别是否启用
func (s *StdLogger) IsPanicEnabled() bool {
	return s.GetLevel() <= PanicLevel
}

// Sync 刷新日志缓冲区
func (s *StdLogger) Sync() error {
	return nil
}

// StdLoggerProvider 标准库log提供者
type StdLoggerProvider struct{}

// NewStdLoggerProvider 创建标准库log提供者
func NewStdLoggerProvider() *StdLoggerProvider {
	return &StdLoggerProvider{}
}

// Create 创建日志实例
func (p *StdLoggerProvider) Create(name string, opts ...Option) Logger {
	return NewStdLogger(name, opts...)
}

// CreateWithConfig 根据配置创建日志实例
func (p *StdLoggerProvider) CreateWithConfig(name string, config map[string]interface{}) Logger {
	return NewStdLogger(name, parseConfigToOptions(config)...)
}
