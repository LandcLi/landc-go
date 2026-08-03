package logger

import (
	"sync"
	"sync/atomic"
)

// LogFactory 日志工厂
type LogFactory struct {
	providers       map[string]LoggerProvider
	defaultProvider string
	mu              sync.RWMutex
}

// 全局日志工厂实例
var (
	factory     *LogFactory
	factoryOnce sync.Once
)

// GetLogFactory 获取全局日志工厂实例
func GetLogFactory() *LogFactory {
	factoryOnce.Do(func() {
		factory = NewLogFactory()
		// 注册默认的日志提供者（无第三方依赖）
		factory.RegisterProvider("console", NewConsoleLoggerProvider())
		factory.RegisterProvider("std", NewStdLoggerProvider())
		// 设置默认提供者为console
		factory.SetDefaultProvider("console")
	})
	return factory
}

// ResetLogFactory 重置全局日志工厂（仅用于测试）
func ResetLogFactory() {
	factoryOnce = sync.Once{}
	factory = nil
	globalLogger.Store((*loggerHolder)(nil))
	loggerInitialized.Store(false)
}

// NewLogFactory 创建日志工厂实例
func NewLogFactory() *LogFactory {
	return &LogFactory{
		providers:       make(map[string]LoggerProvider),
		defaultProvider: "console",
	}
}

// RegisterProvider 注册日志提供者
func (f *LogFactory) RegisterProvider(name string, provider LoggerProvider) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providers[name] = provider
}

// UnregisterProvider 注销日志提供者
func (f *LogFactory) UnregisterProvider(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.providers, name)
}

// SetDefaultProvider 设置默认日志提供者
func (f *LogFactory) SetDefaultProvider(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.defaultProvider = name
}

// GetDefaultProvider 获取默认日志提供者名称
func (f *LogFactory) GetDefaultProvider() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.defaultProvider
}

// GetProvider 获取指定的日志提供者
func (f *LogFactory) GetProvider(name string) (LoggerProvider, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	provider, exists := f.providers[name]
	return provider, exists
}

// CreateLogger 创建日志实例
func (f *LogFactory) CreateLogger(name string) Logger {
	f.mu.RLock()
	defaultProvider := f.defaultProvider
	f.mu.RUnlock()
	return f.CreateLoggerWithProvider(name, defaultProvider)
}

// CreateLoggerWithProvider 使用指定的提供者创建日志实例
func (f *LogFactory) CreateLoggerWithProvider(name, providerName string, opts ...Option) Logger {
	f.mu.RLock()
	provider, exists := f.providers[providerName]
	f.mu.RUnlock()

	if !exists {
		// 如果指定的提供者不存在，使用默认提供者
		f.mu.RLock()
		provider, exists = f.providers[f.defaultProvider]
		f.mu.RUnlock()
		if !exists {
			// 如果默认提供者也不存在，使用控制台日志
			return NewConsoleLogger(name, opts...)
		}
	}

	return provider.Create(name, opts...)
}

// CreateLoggerWithMap 根据配置创建日志实例
func (f *LogFactory) CreateLoggerWithMap(name string, config map[string]interface{}) Logger {
	// 从配置中获取提供者名称
	f.mu.RLock()
	providerName := f.defaultProvider
	f.mu.RUnlock()

	if pn, ok := config["provider"].(string); ok {
		providerName = pn
	}

	f.mu.RLock()
	provider, exists := f.providers[providerName]
	f.mu.RUnlock()

	if !exists {
		// 如果指定的提供者不存在，使用默认提供者
		f.mu.RLock()
		provider, exists = f.providers[f.defaultProvider]
		f.mu.RUnlock()
		if !exists {
			return NewConsoleLogger(name)
		}
	}

	return provider.CreateWithConfig(name, config)
}

// CreateLoggerWithLogConfig 根据LogConfig创建日志实例
func (f *LogFactory) CreateLoggerWithLogConfig(config *LogConfig) Logger {
	// 应用默认值
	config.ApplyDefaults()

	// 获取提供者
	f.mu.RLock()
	provider, exists := f.providers[config.Provider]
	f.mu.RUnlock()

	if !exists {
		// 如果指定的提供者不存在，使用默认提供者
		f.mu.RLock()
		provider, exists = f.providers[f.defaultProvider]
		f.mu.RUnlock()
		if !exists {
			return NewConsoleLogger(config.Name, config.ToOptions()...)
		}
	}

	// 直接使用 ToOptions 创建，避免无意义的中间 map 转换
	return provider.Create(config.Name, config.ToOptions()...)
}

// loggerHolder 用于 atomic.Value 存储 Logger 接口
type loggerHolder struct {
	logger Logger
}

// 全局日志实例（使用 atomic 保证并发安全）
var (
	globalLogger      atomic.Value // 存储 *loggerHolder
	loggerInitialized atomic.Bool
)

func init() {
	// 初始化为 nil holder
	globalLogger.Store((*loggerHolder)(nil))
}

// GetLogger 获取全局日志实例
func GetLogger() Logger {
	// 快速路径：检查是否已有自定义设置的 logger
	if holder, ok := globalLogger.Load().(*loggerHolder); ok && holder != nil {
		return holder.logger
	}

	// 慢速路径：首次初始化
	if loggerInitialized.CompareAndSwap(false, true) {
		l := GetLogFactory().CreateLogger("global")
		globalLogger.Store(&loggerHolder{logger: l})
		return l
	}

	// 其他 goroutine 正在初始化，等待结果
	for {
		if holder, ok := globalLogger.Load().(*loggerHolder); ok && holder != nil {
			return holder.logger
		}
	}
}

// GetLoggerWithName 获取指定名称的日志实例
func GetLoggerWithName(name string) Logger {
	return GetLogFactory().CreateLogger(name)
}

// GetLoggerWithProvider 获取指定提供者的日志实例
func GetLoggerWithProvider(name, provider string, opts ...Option) Logger {
	return GetLogFactory().CreateLoggerWithProvider(name, provider, opts...)
}

// GetLoggerWithMap 根据配置获取日志实例
func GetLoggerWithMap(name string, config map[string]interface{}) Logger {
	return GetLogFactory().CreateLoggerWithMap(name, config)
}

// GetLoggerWithLogConfig 根据LogConfig获取日志实例
func GetLoggerWithLogConfig(config *LogConfig) Logger {
	return GetLogFactory().CreateLoggerWithLogConfig(config)
}

// SetGlobalLogger 设置全局日志实例（并发安全）
func SetGlobalLogger(l Logger) {
	globalLogger.Store(&loggerHolder{logger: l})
	loggerInitialized.Store(true)
}

// Debug 全局调试级日志
func Debug(msg string, fields ...Field) {
	GetLogger().Debug(msg, fields...)
}

// Debugf 全局格式化调试级日志
func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

// Info 全局信息级日志
func Info(msg string, fields ...Field) {
	GetLogger().Info(msg, fields...)
}

// Infof 全局格式化信息级日志
func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

// Warn 全局警告级日志
func Warn(msg string, fields ...Field) {
	GetLogger().Warn(msg, fields...)
}

// Warnf 全局格式化警告级日志
func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

// Error 全局错误级日志
func Error(msg string, fields ...Field) {
	GetLogger().Error(msg, fields...)
}

// Errorf 全局格式化错误级日志
func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

// Fatal 全局致命级日志
func Fatal(msg string, fields ...Field) {
	GetLogger().Fatal(msg, fields...)
}

// Fatalf 全局格式化致命级日志
func Fatalf(format string, args ...interface{}) {
	GetLogger().Fatalf(format, args...)
}

// Panic 全局恐慌级日志
func Panic(msg string, fields ...Field) {
	GetLogger().Panic(msg, fields...)
}

// Panicf 全局格式化恐慌级日志
func Panicf(format string, args ...interface{}) {
	GetLogger().Panicf(format, args...)
}
