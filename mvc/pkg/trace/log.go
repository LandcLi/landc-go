package trace

import (
	"context"

	"github.com/LandcLi/landc-go/log/facade"
)

// WithTraceFields 将链路追踪字段添加到日志
func WithTraceFields(ctx context.Context, fields ...facade.Field) []facade.Field {
	traceID := TraceID(ctx)
	spanID := SpanID(ctx)
	parentSpanID := ParentSpanID(ctx)

	if traceID != "" {
		fields = append(fields, facade.Field{Key: "trace_id", Value: traceID})
	}
	if spanID != "" {
		fields = append(fields, facade.Field{Key: "span_id", Value: spanID})
	}
	if parentSpanID != "" {
		fields = append(fields, facade.Field{Key: "parent_span_id", Value: parentSpanID})
	}

	return fields
}

// WithTraceContext 将链路追踪上下文添加到日志记录器
func WithTraceContext(ctx context.Context, logger facade.Logger) facade.Logger {
	fields := WithTraceFields(ctx)
	return logger.WithFields(fields...)
}

// traceLogger 获取带链路追踪字段的 logger
func traceLogger(ctx context.Context) facade.Logger {
	return facade.GetLogger().WithFields(WithTraceFields(ctx)...)
}

// LogDebug 记录调试级别日志，自动包含链路追踪信息
func LogDebug(ctx context.Context, msg string, fields ...facade.Field) {
	fields = WithTraceFields(ctx, fields...)
	facade.Debug(msg, fields...)
}

// LogInfo 记录信息级别日志，自动包含链路追踪信息
func LogInfo(ctx context.Context, msg string, fields ...facade.Field) {
	fields = WithTraceFields(ctx, fields...)
	facade.Info(msg, fields...)
}

// LogWarn 记录警告级别日志，自动包含链路追踪信息
func LogWarn(ctx context.Context, msg string, fields ...facade.Field) {
	fields = WithTraceFields(ctx, fields...)
	facade.Warn(msg, fields...)
}

// LogError 记录错误级别日志，自动包含链路追踪信息
func LogError(ctx context.Context, msg string, fields ...facade.Field) {
	fields = WithTraceFields(ctx, fields...)
	facade.Error(msg, fields...)
}

// LogFatal 记录致命级别日志，自动包含链路追踪信息
func LogFatal(ctx context.Context, msg string, fields ...facade.Field) {
	fields = WithTraceFields(ctx, fields...)
	facade.Fatal(msg, fields...)
}

// LogPanic 记录恐慌级别日志，自动包含链路追踪信息
func LogPanic(ctx context.Context, msg string, fields ...facade.Field) {
	fields = WithTraceFields(ctx, fields...)
	facade.Panic(msg, fields...)
}

// LogDebugf 记录格式化调试级别日志，自动包含链路追踪信息
func LogDebugf(ctx context.Context, format string, args ...interface{}) {
	traceLogger(ctx).Debugf(format, args...)
}

// LogInfof 记录格式化信息级别日志，自动包含链路追踪信息
func LogInfof(ctx context.Context, format string, args ...interface{}) {
	traceLogger(ctx).Infof(format, args...)
}

// LogWarnf 记录格式化警告级别日志，自动包含链路追踪信息
func LogWarnf(ctx context.Context, format string, args ...interface{}) {
	traceLogger(ctx).Warnf(format, args...)
}

// LogErrorf 记录格式化错误级别日志，自动包含链路追踪信息
func LogErrorf(ctx context.Context, format string, args ...interface{}) {
	traceLogger(ctx).Errorf(format, args...)
}

// LogFatalf 记录格式化致命级别日志，自动包含链路追踪信息
func LogFatalf(ctx context.Context, format string, args ...interface{}) {
	traceLogger(ctx).Fatalf(format, args...)
}

// LogPanicf 记录格式化恐慌级别日志，自动包含链路追踪信息
func LogPanicf(ctx context.Context, format string, args ...interface{}) {
	traceLogger(ctx).Panicf(format, args...)
}
