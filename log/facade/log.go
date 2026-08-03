// Package facade 提供 landc-go 日志门面的公开 API，支持 Zap/Logrus/Console 等多种后端。
package facade

import (
	"time"

	"github.com/LandcLi/landc-go/log/internal/logger"
)

// 核心类型和函数导出

// LogLevel 定义日志级别，支持 Debug、Info、Warn、Error、Fatal、Panic 六个级别
type LogLevel = logger.LogLevel

// Field 定义日志字段，用于结构化日志
type Field = logger.Field

// Logger 日志门面接口，定义了统一的日志方法
type Logger = logger.Logger

// Option 日志配置选项，用于创建日志实例时的配置
type Option = logger.Option

// LoggerOptions 日志配置选项集合，用于存储应用后的配置
type LoggerOptions = logger.LoggerOptions

// LogConfig 统一的日志配置类，包含所有日志提供者的配置项
type LogConfig = logger.LogConfig

// LoggerProvider 日志提供者接口
type LoggerProvider = logger.LoggerProvider

// 日志级别常量
const (
	// DebugLevel 调试级别日志
	DebugLevel LogLevel = logger.DebugLevel
	// InfoLevel 信息级别日志
	InfoLevel LogLevel = logger.InfoLevel
	// WarnLevel 警告级别日志
	WarnLevel LogLevel = logger.WarnLevel
	// ErrorLevel 错误级别日志
	ErrorLevel LogLevel = logger.ErrorLevel
	// FatalLevel 致命级别日志，输出后程序会退出
	FatalLevel LogLevel = logger.FatalLevel
	// PanicLevel 恐慌级别日志，输出后会触发 panic
	PanicLevel LogLevel = logger.PanicLevel
)

// GetLogger 获取全局日志实例
// 全局日志实例是一个默认的日志实例，可直接使用
func GetLogger() Logger {
	return logger.GetLogger()
}

// GetLoggerWithName 获取指定名称的日志实例
// name: 日志实例名称
func GetLoggerWithName(name string) Logger {
	return logger.GetLoggerWithName(name)
}

// GetLoggerWithProvider 获取指定提供者的日志实例
// name: 日志实例名称
// provider: 日志提供者名称，如 "console"、"zap"、"logrus"、"std"
// opts: 日志配置选项
func GetLoggerWithProvider(name, provider string, opts ...Option) Logger {
	return logger.GetLoggerWithProvider(name, provider, opts...)
}

// GetLoggerWithMap 根据配置map获取日志实例
// name: 日志实例名称
// config: 配置map，包含各种配置项
func GetLoggerWithMap(name string, config map[string]interface{}) Logger {
	return logger.GetLoggerWithMap(name, config)
}

// GetLoggerWithLogConfig 根据LogConfig获取日志实例
// config: 统一的日志配置对象
func GetLoggerWithLogConfig(config *LogConfig) Logger {
	return logger.GetLoggerWithLogConfig(config)
}

// SetGlobalLogger 设置全局日志实例
// log: 要设置的日志实例
func SetGlobalLogger(log Logger) {
	logger.SetGlobalLogger(log)
}

// NewLogConfig 创建默认的日志配置
// 返回一个带有默认值的LogConfig实例
func NewLogConfig() *LogConfig {
	return logger.NewLogConfig()
}

// 配置选项函数

// WithLevel 设置日志级别
// level: 日志级别，如 DebugLevel、InfoLevel 等
func WithLevel(level LogLevel) Option {
	return logger.WithLevel(level)
}

// WithFormat 设置日志格式
// format: 日志格式，支持 "text" 和 "json"
func WithFormat(format string) Option {
	return logger.WithFormat(format)
}

// WithOutputPath 设置日志输出路径
// path: 日志文件路径，如 "./logs/app.log"
func WithOutputPath(path string) Option {
	return logger.WithOutputPath(path)
}

// WithConfig 设置额外配置
// config: 额外配置map，用于传递特定日志提供者的配置
func WithConfig(config map[string]interface{}) Option {
	return logger.WithConfig(config)
}

// WithMaxLogSize 设置单个日志文件最大大小（MB）
// size: 日志文件最大大小，单位为MB
func WithMaxLogSize(size int64) Option {
	return logger.WithMaxLogSize(size)
}

// WithMaxLogAge 设置日志文件最大保留时间
// age: 日志文件最大保留时间
func WithMaxLogAge(age time.Duration) Option {
	return logger.WithMaxLogAge(age)
}

// WithMaxLogFiles 设置最大保留日志文件数量
// files: 最大保留日志文件数量
func WithMaxLogFiles(files int) Option {
	return logger.WithMaxLogFiles(files)
}

// WithCompressLogs 设置是否压缩旧日志
// compress: 是否压缩旧日志
func WithCompressLogs(compress bool) Option {
	return logger.WithCompressLogs(compress)
}

// WithMaxMessageSize 设置单条日志最大大小（KB）
// size: 单条日志最大大小，单位为KB
func WithMaxMessageSize(size int) Option {
	return logger.WithMaxMessageSize(size)
}

// 全局日志函数

// Debug 全局调试级日志
// msg: 日志消息
// fields: 日志字段
func Debug(msg string, fields ...Field) {
	logger.Debug(msg, fields...)
}

// Debugf 全局格式化调试级日志
// format: 格式化字符串
// args: 格式化参数
func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

// Info 全局信息级日志
// msg: 日志消息
// fields: 日志字段
func Info(msg string, fields ...Field) {
	logger.Info(msg, fields...)
}

// Infof 全局格式化信息级日志
// format: 格式化字符串
// args: 格式化参数
func Infof(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

// Warn 全局警告级日志
// msg: 日志消息
// fields: 日志字段
func Warn(msg string, fields ...Field) {
	logger.Warn(msg, fields...)
}

// Warnf 全局格式化警告级日志
// format: 格式化字符串
// args: 格式化参数
func Warnf(format string, args ...interface{}) {
	logger.Warnf(format, args...)
}

// Error 全局错误级日志
// msg: 日志消息
// fields: 日志字段
func Error(msg string, fields ...Field) {
	logger.Error(msg, fields...)
}

// Errorf 全局格式化错误级日志
// format: 格式化字符串
// args: 格式化参数
func Errorf(format string, args ...interface{}) {
	logger.Errorf(format, args...)
}

// Fatal 全局致命级日志，输出后程序会退出
// msg: 日志消息
// fields: 日志字段
func Fatal(msg string, fields ...Field) {
	logger.Fatal(msg, fields...)
}

// Fatalf 全局格式化致命级日志，输出后程序会退出
// format: 格式化字符串
// args: 格式化参数
func Fatalf(format string, args ...interface{}) {
	logger.Fatalf(format, args...)
}

// Panic 全局恐慌级日志，输出后会触发 panic
// msg: 日志消息
// fields: 日志字段
func Panic(msg string, fields ...Field) {
	logger.Panic(msg, fields...)
}

// Panicf 全局格式化恐慌级日志，输出后会触发 panic
// format: 格式化字符串
// args: 格式化参数
func Panicf(format string, args ...interface{}) {
	logger.Panicf(format, args...)
}

// RegisterProvider 注册日志提供者
// name: 提供者名称
// provider: 日志提供者实例
func RegisterProvider(name string, provider LoggerProvider) {
	logger.GetLogFactory().RegisterProvider(name, provider)
}

// UnregisterProvider 注销日志提供者
// name: 提供者名称
func UnregisterProvider(name string) {
	logger.GetLogFactory().UnregisterProvider(name)
}
