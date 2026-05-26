package tests

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/log/facade"
)

// TestLoggerInterface 测试Logger接口的基本功能
func TestLoggerInterface(t *testing.T) {
	// 测试控制台日志
	logger := facade.GetLoggerWithProvider("test", "console")

	// 测试日志级别设置
	logger.SetLevel(facade.DebugLevel)
	if logger.GetLevel() != facade.DebugLevel {
		t.Errorf("Expected level DebugLevel, got %v", logger.GetLevel())
	}

	// 测试日志输出（这里只是测试方法调用，不测试输出内容）
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warn message")
	logger.Error("Error message")

	// 测试格式化日志
	logger.Debugf("Debug: %s", "test")
	logger.Infof("Info: %s", "test")
	logger.Warnf("Warn: %s", "test")
	logger.Errorf("Error: %s", "test")

	// 测试字段
	logger.Info("With fields",
		facade.Field{Key: "key1", Value: "value1"},
		facade.Field{Key: "key2", Value: 123},
	)

	// 测试链式调用
	logger.WithField("chain", "value").Info("Chained logger")

	// 测试上下文
	ctx := context.Background()
	logger.WithContext(ctx).Info("With context")

	// 测试错误
	err := errors.New("test error")
	logger.WithError(err).Error("With error")

	// 测试时间
	now := time.Now()
	logger.WithTime(now).Info("With time")

	// 测试日志级别检查
	if !logger.IsDebugEnabled() {
		t.Error("Expected DebugLevel to be enabled")
	}
	if !logger.IsInfoEnabled() {
		t.Error("Expected InfoLevel to be enabled")
	}
	if !logger.IsWarnEnabled() {
		t.Error("Expected WarnLevel to be enabled")
	}
	if !logger.IsErrorEnabled() {
		t.Error("Expected ErrorLevel to be enabled")
	}

	// 测试Sync
	if err := logger.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
	}
}

// TestGlobalLogger 测试全局日志
func TestGlobalLogger(t *testing.T) {
	// 测试获取全局日志实例
	globalLogger := facade.GetLogger()
	if globalLogger == nil {
		t.Error("Failed to get global logger")
	}

	// 测试全局日志函数
	facade.Debug("Global debug")
	facade.Info("Global info")
	facade.Warn("Global warn")
	facade.Error("Global error")

	// 测试全局格式化函数
	facade.Debugf("Global debug: %s", "test")
	facade.Infof("Global info: %s", "test")
	facade.Warnf("Global warn: %s", "test")
	facade.Errorf("Global error: %s", "test")

	// 测试全局函数带字段
	facade.Info("Global with fields", facade.Field{Key: "key", Value: "value"})
}

// TestZapLogger 测试zap日志适配器
func TestZapLogger(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test-zap", "zap")
	logger.SetLevel(facade.DebugLevel)

	if logger == nil {
		t.Error("Failed to create zap logger")
	}

	// 测试zap日志功能
	logger.Debug("Zap debug")
	logger.Info("Zap info")
	logger.Warn("Zap warn")
	logger.Error("Zap error")

	// 测试字段
	logger.Info("Zap with fields",
		facade.Field{Key: "zap", Value: "test"},
	)

	// 测试Sync
	if err := logger.Sync(); err != nil {
		t.Errorf("Zap sync failed: %v", err)
	}
}

// TestLogrusLogger 测试logrus日志适配器
func TestLogrusLogger(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test-logrus", "logrus")
	logger.SetLevel(facade.DebugLevel)

	if logger == nil {
		t.Error("Failed to create logrus logger")
	}

	// 测试logrus日志功能
	logger.Debug("Logrus debug")
	logger.Info("Logrus info")
	logger.Warn("Logrus warn")
	logger.Error("Logrus error")

	// 测试字段
	logger.Info("Logrus with fields",
		facade.Field{Key: "logrus", Value: "test"},
	)

	// 测试Sync
	if err := logger.Sync(); err != nil {
		t.Errorf("Logrus sync failed: %v", err)
	}
}

// TestStdLogger 测试标准库日志适配器
func TestStdLogger(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test-std", "std")
	logger.SetLevel(facade.DebugLevel)

	if logger == nil {
		t.Error("Failed to create std logger")
	}

	// 测试std日志功能
	logger.Debug("Std debug")
	logger.Info("Std info")
	logger.Warn("Std warn")
	logger.Error("Std error")

	// 测试字段
	logger.Info("Std with fields",
		facade.Field{Key: "std", Value: "test"},
	)

	// 测试Sync
	if err := logger.Sync(); err != nil {
		t.Errorf("Std sync failed: %v", err)
	}
}

// TestConsoleLogger 测试控制台日志适配器
func TestConsoleLogger(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test-console", "console")
	logger.SetLevel(facade.DebugLevel)

	if logger == nil {
		t.Error("Failed to create console logger")
	}

	// 测试console日志功能
	logger.Debug("Console debug")
	logger.Info("Console info")
	logger.Warn("Console warn")
	logger.Error("Console error")

	// 测试字段
	logger.Info("Console with fields",
		facade.Field{Key: "console", Value: "test"},
	)

	// 测试Sync
	if err := logger.Sync(); err != nil {
		t.Errorf("Console sync failed: %v", err)
	}
}

// TestLoggerOptions 测试日志选项
func TestLoggerOptions(t *testing.T) {
	// 测试选项函数
	levelOpt := facade.WithLevel(facade.DebugLevel)
	formatOpt := facade.WithFormat("json")
	outputOpt := facade.WithOutputPath("stdout")
	configOpt := facade.WithConfig(map[string]interface{}{"key": "value"})

	options := &facade.LoggerOptions{}

	// 应用选项
	levelOpt(options)
	formatOpt(options)
	outputOpt(options)
	configOpt(options)

	// 验证选项
	if options.Level != facade.DebugLevel {
		t.Errorf("Expected level DebugLevel, got %v", options.Level)
	}

	if options.Format != "json" {
		t.Errorf("Expected format 'json', got '%s'", options.Format)
	}

	if options.OutputPath != "stdout" {
		t.Errorf("Expected output path 'stdout', got '%s'", options.OutputPath)
	}

	if options.Config["key"] != "value" {
		t.Errorf("Expected config key 'key' to be 'value', got '%v'", options.Config["key"])
	}
}

// TestLogLevelString 测试日志级别字符串表示
func TestLogLevelString(t *testing.T) {
	testCases := []struct {
		level    facade.LogLevel
		expected string
	}{
		{facade.DebugLevel, "DEBUG"},
		{facade.InfoLevel, "INFO"},
		{facade.WarnLevel, "WARN"},
		{facade.ErrorLevel, "ERROR"},
		{facade.FatalLevel, "FATAL"},
		{facade.PanicLevel, "PANIC"},
		{facade.LogLevel(999), "UNKNOWN"},
	}

	for _, tc := range testCases {
		if tc.level.String() != tc.expected {
			t.Errorf("Expected level %v to be '%s', got '%s'", tc.level, tc.expected, tc.level.String())
		}
	}
}

// TestWithMethods 测试With系列方法
func TestWithMethods(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test-with", "console")

	// 测试WithField
	logger1 := logger.WithField("key1", "value1")
	if logger1 == nil {
		t.Error("Failed to create logger with field")
	}

	// 测试WithFields
	logger2 := logger.WithFields(
		facade.Field{Key: "key1", Value: "value1"},
		facade.Field{Key: "key2", Value: "value2"},
	)
	if logger2 == nil {
		t.Error("Failed to create logger with fields")
	}

	// 测试WithContext
	ctx := context.Background()
	logger3 := logger.WithContext(ctx)
	if logger3 == nil {
		t.Error("Failed to create logger with context")
	}

	// 测试WithError
	err := errors.New("test error")
	logger4 := logger.WithError(err)
	if logger4 == nil {
		t.Error("Failed to create logger with error")
	}

	// 测试WithTime
	now := time.Now()
	logger5 := logger.WithTime(now)
	if logger5 == nil {
		t.Error("Failed to create logger with time")
	}
}

// TestIsEnabledMethods 测试IsEnabled系列方法
func TestIsEnabledMethods(t *testing.T) {
	// 测试DebugLevel
	debugLogger := facade.GetLoggerWithProvider("test-debug", "console")
	debugLogger.SetLevel(facade.DebugLevel)
	if !debugLogger.IsDebugEnabled() {
		t.Error("Debug level should be enabled")
	}
	if !debugLogger.IsInfoEnabled() {
		t.Error("Info level should be enabled")
	}
	if !debugLogger.IsWarnEnabled() {
		t.Error("Warn level should be enabled")
	}
	if !debugLogger.IsErrorEnabled() {
		t.Error("Error level should be enabled")
	}

	// 测试InfoLevel
	infoLogger := facade.GetLoggerWithProvider("test-info", "console")
	infoLogger.SetLevel(facade.InfoLevel)
	if infoLogger.IsDebugEnabled() {
		t.Error("Debug level should not be enabled")
	}
	if !infoLogger.IsInfoEnabled() {
		t.Error("Info level should be enabled")
	}

	// 测试ErrorLevel
	errorLogger := facade.GetLoggerWithProvider("test-error", "console")
	errorLogger.SetLevel(facade.ErrorLevel)
	if errorLogger.IsDebugEnabled() {
		t.Error("Debug level should not be enabled")
	}
	if errorLogger.IsInfoEnabled() {
		t.Error("Info level should not be enabled")
	}
	if errorLogger.IsWarnEnabled() {
		t.Error("Warn level should not be enabled")
	}
	if !errorLogger.IsErrorEnabled() {
		t.Error("Error level should be enabled")
	}
}

// TestConcurrentSetLevel 测试并发设置日志级别（验证 data race 修复）
func TestConcurrentSetLevel(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test-concurrent", "console")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			logger.SetLevel(facade.DebugLevel)
		}()
		go func() {
			defer wg.Done()
			_ = logger.GetLevel()
		}()
	}
	wg.Wait()
}

// TestConcurrentSetLevelZap 测试 Zap 并发设置日志级别
func TestConcurrentSetLevelZap(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test-concurrent-zap", "zap")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			logger.SetLevel(facade.DebugLevel)
			logger.Debug("concurrent debug")
		}()
		go func() {
			defer wg.Done()
			logger.SetLevel(facade.ErrorLevel)
			logger.Error("concurrent error")
		}()
	}
	wg.Wait()
}

// TestConcurrentSetLevelLogrus 测试 Logrus 并发设置日志级别
func TestConcurrentSetLevelLogrus(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test-concurrent-logrus", "logrus")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			logger.SetLevel(facade.DebugLevel)
			logger.Debug("concurrent debug")
		}()
		go func() {
			defer wg.Done()
			logger.SetLevel(facade.ErrorLevel)
			logger.Error("concurrent error")
		}()
	}
	wg.Wait()
}

// TestWithFieldsSliceIsolation 测试 WithFields 不会污染父 logger 的字段
func TestWithFieldsSliceIsolation(t *testing.T) {
	parent := facade.GetLoggerWithProvider("test-isolation", "console")
	parent.SetLevel(facade.DebugLevel)

	// 创建子 logger
	child1 := parent.WithField("child", "1")
	child2 := parent.WithField("child", "2")

	// 父 logger 不应该被子 logger 影响
	parent.Info("parent should have no child fields")
	child1.Info("child1")
	child2.Info("child2")

	// 再在 child1 上追加字段，不应该影响 child2
	grandchild := child1.WithField("grandchild", "gc")
	grandchild.Info("grandchild")
	child1.Info("child1 should not have grandchild field")
}

// TestSetGlobalLoggerConcurrent 测试并发设置全局 logger
func TestSetGlobalLoggerConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			l := facade.GetLoggerWithProvider("custom-global", "console")
			facade.SetGlobalLogger(l)
		}()
		go func() {
			defer wg.Done()
			l := facade.GetLogger()
			if l == nil {
				t.Error("GetLogger returned nil")
			}
		}()
	}
	wg.Wait()
}

// TestZapSetLevelEffective 测试 Zap SetLevel 实际生效
func TestZapSetLevelEffective(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test-zap-level", "zap",
		facade.WithLevel(facade.ErrorLevel),
		facade.WithFormat("json"),
	)

	// 设置为 Error 级别，Debug 应该被过滤
	if logger.IsDebugEnabled() {
		t.Error("Debug should not be enabled at ErrorLevel")
	}

	// 动态调整为 Debug 级别
	logger.SetLevel(facade.DebugLevel)
	if !logger.IsDebugEnabled() {
		t.Error("Debug should be enabled after SetLevel(DebugLevel)")
	}

	// 调用 Debug 不应 panic（如果 SetLevel 未真正生效会被 zap 过滤但不会 panic）
	logger.Debug("this should work after SetLevel")
}

// TestLogrusFormatInjectionSafe 测试 Logrus 格式注入安全性
func TestLogrusFormatInjectionSafe(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test-logrus-safe", "logrus",
		facade.WithLevel(facade.DebugLevel),
	)

	// 包含 %s %d 等格式化动词的消息不应导致 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logrus format injection caused panic: %v", r)
		}
	}()

	logger.Debugf("normal: %s %d", "hello", 42)

	// 先格式化再输出的场景：消息本身含 % 符号
	msgWithPercent := "progress: 50% done, status=%s"
	logger.Debug(msgWithPercent)
	logger.Info(msgWithPercent)
}

// TestMultiByteMessageTruncation 测试多字节字符截断安全性
func TestMultiByteMessageTruncation(t *testing.T) {
	logger := facade.GetLoggerWithProvider("test-truncate", "console",
		facade.WithLevel(facade.DebugLevel),
		facade.WithMaxMessageSize(1), // 1KB 限制
	)

	// 创建一个包含中文字符的长消息（超过 1KB）
	longMsg := strings.Repeat("你好世界", 200) // 每个中文字符 3 字节，共 2400 字节
	logger.Info(longMsg)

	// 不应 panic，这就是成功
}

// TestLogConfigApplyDefaults 测试 LogConfig 默认值应用
func TestLogConfigApplyDefaults(t *testing.T) {
	config := &facade.LogConfig{}
	config.ApplyDefaults()

	if config.Provider != "console" {
		t.Errorf("Expected provider 'console', got '%s'", config.Provider)
	}
	if config.Name != "app" {
		t.Errorf("Expected name 'app', got '%s'", config.Name)
	}
	if config.Format != "text" {
		t.Errorf("Expected format 'text', got '%s'", config.Format)
	}
	if config.OutputPath != "stdout" {
		t.Errorf("Expected outputPath 'stdout', got '%s'", config.OutputPath)
	}
	if config.MaxLogSize != 100 {
		t.Errorf("Expected maxLogSize 100, got %d", config.MaxLogSize)
	}
	if config.MaxLogFiles != 10 {
		t.Errorf("Expected maxLogFiles 10, got %d", config.MaxLogFiles)
	}

	// 无效格式应被修正为 text
	config2 := &facade.LogConfig{Format: "invalid"}
	config2.ApplyDefaults()
	if config2.Format != "text" {
		t.Errorf("Expected invalid format to be corrected to 'text', got '%s'", config2.Format)
	}
}

// TestLogConfigBackwardCompatibility 测试 Validate 向后兼容
func TestLogConfigBackwardCompatibility(t *testing.T) {
	config := facade.NewLogConfig()
	result := config.Validate()
	if !result {
		t.Error("Validate should return true for backward compatibility")
	}
}

// TestLogConfigChainedBuilder 测试链式配置构建
func TestLogConfigChainedBuilder(t *testing.T) {
	config := facade.NewLogConfig()
	config.WithProvider("console").
		WithName("test-app").
		WithLevel(facade.DebugLevel).
		WithFormat("json").
		WithOutputPath("stdout").
		WithMaxLogSize(50).
		WithMaxLogAge(24 * time.Hour).
		WithMaxLogFiles(5).
		WithCompressLogs(true).
		WithMaxMessageSize(10).
		WithExtraConfig("key1", "value1")

	if config.Provider != "console" {
		t.Errorf("Expected provider 'console', got '%s'", config.Provider)
	}
	if config.Name != "test-app" {
		t.Errorf("Expected name 'test-app', got '%s'", config.Name)
	}
	if config.Level != facade.DebugLevel {
		t.Errorf("Expected level DebugLevel, got %v", config.Level)
	}
	if config.MaxLogSize != 50 {
		t.Errorf("Expected maxLogSize 50, got %d", config.MaxLogSize)
	}

	// 测试 GetLoggerWithLogConfig
	logger := facade.GetLoggerWithLogConfig(config)
	if logger == nil {
		t.Fatal("Failed to create logger with LogConfig")
	}
	logger.Debug("test LogConfig logger")
}

// TestConcurrentLogOutput 测试并发日志输出
func TestConcurrentLogOutput(t *testing.T) {
	providers := []string{"console", "std", "zap", "logrus"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			logger := facade.GetLoggerWithProvider("test-concurrent-output", provider,
				facade.WithLevel(facade.DebugLevel),
			)

			var wg sync.WaitGroup
			for i := 0; i < 50; i++ {
				wg.Add(1)
				go func(n int) {
					defer wg.Done()
					logger.Infof("concurrent log #%d", n)
					logger.WithField("goroutine", n).Debug("with field")
				}(i)
			}
			wg.Wait()
		})
	}
}
