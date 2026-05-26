package logger

import (
	"context"
	"time"
	"unicode/utf8"
)

// LogLevel 定义日志级别
type LogLevel int32

const (
	// DebugLevel 调试级别
	DebugLevel LogLevel = iota
	// InfoLevel 信息级别
	InfoLevel
	// WarnLevel 警告级别
	WarnLevel
	// ErrorLevel 错误级别
	ErrorLevel
	// FatalLevel 致命级别
	FatalLevel
	// PanicLevel 恐慌级别
	PanicLevel
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	case PanicLevel:
		return "PANIC"
	default:
		return "UNKNOWN"
	}
}

// Field 定义日志字段
type Field struct {
	Key   string
	Value interface{}
}

// Logger 日志门面接口
type Logger interface {
	// SetLevel 设置日志级别
	SetLevel(level LogLevel)
	// GetLevel 获取当前日志级别
	GetLevel() LogLevel

	// Debug 输出调试级日志
	Debug(msg string, fields ...Field)
	// Debugf 输出格式化的调试级日志
	Debugf(format string, args ...interface{})

	// Info 输出信息级日志
	Info(msg string, fields ...Field)
	// Infof 输出格式化的信息级日志
	Infof(format string, args ...interface{})

	// Warn 输出警告级日志
	Warn(msg string, fields ...Field)
	// Warnf 输出格式化的警告级日志
	Warnf(format string, args ...interface{})

	// Error 输出错误级日志
	Error(msg string, fields ...Field)
	// Errorf 输出格式化的错误级日志
	Errorf(format string, args ...interface{})

	// Fatal 输出致命级日志并退出程序
	Fatal(msg string, fields ...Field)
	// Fatalf 输出格式化的致命级日志并退出程序
	Fatalf(format string, args ...interface{})

	// Panic 输出恐慌级日志并触发panic
	Panic(msg string, fields ...Field)
	// Panicf 输出格式化的恐慌级日志并触发panic
	Panicf(format string, args ...interface{})

	// WithFields 添加字段到日志
	WithFields(fields ...Field) Logger
	// WithField 添加单个字段到日志
	WithField(key string, value interface{}) Logger
	// WithContext 添加上下文到日志
	WithContext(ctx context.Context) Logger
	// WithError 添加错误信息到日志
	WithError(err error) Logger
	// WithTime 添加时间到日志
	WithTime(t time.Time) Logger

	// IsDebugEnabled 检查调试级别是否启用
	IsDebugEnabled() bool
	// IsInfoEnabled 检查信息级别是否启用
	IsInfoEnabled() bool
	// IsWarnEnabled 检查警告级别是否启用
	IsWarnEnabled() bool
	// IsErrorEnabled 检查错误级别是否启用
	IsErrorEnabled() bool
	// IsFatalEnabled 检查致命级别是否启用
	IsFatalEnabled() bool
	// IsPanicEnabled 检查恐慌级别是否启用
	IsPanicEnabled() bool

	// Sync 刷新日志缓冲区
	Sync() error
}

// LoggerProvider 日志提供者接口
type LoggerProvider interface {
	// Create 创建日志实例
	Create(name string, opts ...Option) Logger
	// CreateWithConfig 根据配置创建日志实例
	CreateWithConfig(name string, config map[string]interface{}) Logger
}

// Option 日志配置选项
type Option func(*LoggerOptions)

// LoggerOptions 日志配置选项
type LoggerOptions struct {
	Level          LogLevel
	Format         string
	OutputPath     string
	MaxLogSize     int64         // 单个日志文件最大大小（MB）
	MaxLogAge      time.Duration // 日志文件最大保留时间
	MaxLogFiles    int           // 最大保留日志文件数量
	CompressLogs   bool          // 是否压缩旧日志
	MaxMessageSize int           // 单条日志最大大小（KB）
	Config         map[string]interface{}
}

// DefaultLoggerOptions 返回默认的日志配置选项
func DefaultLoggerOptions() *LoggerOptions {
	return &LoggerOptions{
		Level:          InfoLevel,
		Format:         "text",
		OutputPath:     "stdout",
		MaxLogSize:     100,
		MaxLogAge:      7 * 24 * time.Hour,
		MaxLogFiles:    10,
		CompressLogs:   false,
		MaxMessageSize: 0,
		Config:         make(map[string]interface{}),
	}
}

// WithLevel 设置日志级别
func WithLevel(level LogLevel) Option {
	return func(opt *LoggerOptions) {
		opt.Level = level
	}
}

// WithFormat 设置日志格式
func WithFormat(format string) Option {
	return func(opt *LoggerOptions) {
		opt.Format = format
	}
}

// WithOutputPath 设置日志输出路径
func WithOutputPath(path string) Option {
	return func(opt *LoggerOptions) {
		opt.OutputPath = path
	}
}

// WithConfig 设置额外配置
func WithConfig(config map[string]interface{}) Option {
	return func(opt *LoggerOptions) {
		opt.Config = config
	}
}

// WithMaxLogSize 设置单个日志文件最大大小（MB）
func WithMaxLogSize(size int64) Option {
	return func(opt *LoggerOptions) {
		opt.MaxLogSize = size
	}
}

// WithMaxLogAge 设置日志文件最大保留时间
func WithMaxLogAge(age time.Duration) Option {
	return func(opt *LoggerOptions) {
		opt.MaxLogAge = age
	}
}

// WithMaxLogFiles 设置最大保留日志文件数量
func WithMaxLogFiles(files int) Option {
	return func(opt *LoggerOptions) {
		opt.MaxLogFiles = files
	}
}

// WithCompressLogs 设置是否压缩旧日志
func WithCompressLogs(compress bool) Option {
	return func(opt *LoggerOptions) {
		opt.CompressLogs = compress
	}
}

// WithMaxMessageSize 设置单条日志最大大小（KB）
func WithMaxMessageSize(size int) Option {
	return func(opt *LoggerOptions) {
		opt.MaxMessageSize = size
	}
}

// CopyFields 安全复制字段切片，避免 append 共享底层数组问题
func CopyFields(src []Field) []Field {
	if src == nil {
		return nil
	}
	dst := make([]Field, len(src))
	copy(dst, src)
	return dst
}

// MergeFields 安全合并两组字段，返回新切片
func MergeFields(base []Field, extra []Field) []Field {
	result := make([]Field, 0, len(base)+len(extra))
	result = append(result, base...)
	result = append(result, extra...)
	return result
}

// SafeLimitMessageSize 限制日志消息大小，安全处理多字节字符
func SafeLimitMessageSize(msg string, maxMessageSizeKB int) string {
	if maxMessageSizeKB <= 0 {
		return msg
	}
	maxSize := maxMessageSizeKB * 1024
	if len(msg) <= maxSize {
		return msg
	}
	// 安全截断：确保不会截断 UTF-8 多字节字符的中间
	truncated := msg[:maxSize-3]
	// 如果最后一个字符是不完整的 UTF-8 序列，向前回退
	for len(truncated) > 0 && !utf8.ValidString(truncated) {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated + "..."
}
