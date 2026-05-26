package trace

import (
	"context"
	"fmt"
	"time"

	"github.com/LandcLi/landc-go/tools/generate"
)

type contextKey string

var (
	traceIDKey    = contextKey("traceID")
	spanIDKey     = contextKey("spanID")
	parentSpanKey = contextKey("parentSpanID")
)

// TraceID 从上下文中获取追踪ID
func TraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// SpanID 从上下文中获取跨度ID
func SpanID(ctx context.Context) string {
	if spanID, ok := ctx.Value(spanIDKey).(string); ok {
		return spanID
	}
	return ""
}

// ParentSpanID 从上下文中获取父跨度ID
func ParentSpanID(ctx context.Context) string {
	if parentSpanID, ok := ctx.Value(parentSpanKey).(string); ok {
		return parentSpanID
	}
	return ""
}

// WithTraceID 将追踪ID添加到上下文中
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// WithSpanID 将跨度ID添加到上下文中
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, spanIDKey, spanID)
}

// WithParentSpanID 将父跨度ID添加到上下文中
func WithParentSpanID(ctx context.Context, parentSpanID string) context.Context {
	return context.WithValue(ctx, parentSpanKey, parentSpanID)
}

// NewTraceID 生成新的追踪ID（使用 landc-go/tools/generate）
func NewTraceID() string {
	return generate.UUID()
}

// NewSpanID 生成新的跨度ID
func NewSpanID() string {
	return generate.UUID()
}

// InitTrace 初始化追踪上下文
func InitTrace(ctx context.Context) context.Context {
	traceID := NewTraceID()
	spanID := NewSpanID()

	ctx = WithTraceID(ctx, traceID)
	ctx = WithSpanID(ctx, spanID)

	return ctx
}

// InitTraceWithParent 使用父跨度初始化追踪上下文
func InitTraceWithParent(ctx context.Context, parentTraceID, parentSpanID string) context.Context {
	traceID := parentTraceID
	if traceID == "" {
		traceID = NewTraceID()
	}
	spanID := NewSpanID()

	ctx = WithTraceID(ctx, traceID)
	ctx = WithSpanID(ctx, spanID)
	ctx = WithParentSpanID(ctx, parentSpanID)

	return ctx
}

// NewSpan 创建新的跨度
func NewSpan(ctx context.Context, name string) (context.Context, *Span) {
	parentTraceID := TraceID(ctx)
	parentSpanID := SpanID(ctx)

	traceID := parentTraceID
	if traceID == "" {
		traceID = NewTraceID()
	}
	spanID := NewSpanID()

	newCtx := WithTraceID(ctx, traceID)
	newCtx = WithSpanID(newCtx, spanID)
	if parentSpanID != "" {
		newCtx = WithParentSpanID(newCtx, parentSpanID)
	}

	span := &Span{
		Name:         name,
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		StartTime:    time.Now(),
	}

	return newCtx, span
}

// Span 跨度信息
type Span struct {
	Name         string
	TraceID      string
	SpanID       string
	ParentSpanID string
	StartTime    time.Time
	EndTime      time.Time
	Success      bool
	Error        error
}

// End 结束跨度
func (s *Span) End() {
	s.EndTime = time.Now()
	s.Success = s.Error == nil
}

// EndWithError 结束跨度并记录错误
func (s *Span) EndWithError(err error) {
	s.Error = err
	s.End()
}

// Duration 获取跨度持续时间
func (s *Span) Duration() time.Duration {
	return s.EndTime.Sub(s.StartTime)
}

// String 返回跨度的字符串表示
func (s *Span) String() string {
	return fmt.Sprintf("Span{Name: %s, TraceID: %s, SpanID: %s, ParentSpanID: %s, Duration: %v, Success: %v}",
		s.Name, s.TraceID, s.SpanID, s.ParentSpanID, s.Duration(), s.Success)
}

// GetTraceInfo 获取追踪信息
func GetTraceInfo(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"trace_id":       TraceID(ctx),
		"span_id":        SpanID(ctx),
		"parent_span_id": ParentSpanID(ctx),
	}
}
