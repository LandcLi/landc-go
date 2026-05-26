package trace_test

import (
	"context"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/mvc/pkg/trace"
)

func TestInitTrace(t *testing.T) {
	ctx := context.Background()

	// 初始化追踪
	ctx = trace.InitTrace(ctx)

	// 检查 TraceID
	traceID := trace.TraceID(ctx)
	if traceID == "" {
		t.Error("Expected non-empty trace ID")
	}

	// 检查 SpanID
	spanID := trace.SpanID(ctx)
	if spanID == "" {
		t.Error("Expected non-empty span ID")
	}

	// 检查 ParentSpanID（应该为空）
	parentSpanID := trace.ParentSpanID(ctx)
	if parentSpanID != "" {
		t.Errorf("Expected empty parent span ID, got %s", parentSpanID)
	}
}

func TestInitTraceWithParent(t *testing.T) {
	ctx := context.Background()

	// 使用父跨度初始化追踪
	parentTraceID := "parent-trace-id"
	parentSpanID := "parent-span-id"
	ctx = trace.InitTraceWithParent(ctx, parentTraceID, parentSpanID)

	// 检查 TraceID
	traceID := trace.TraceID(ctx)
	if traceID != parentTraceID {
		t.Errorf("Expected trace ID %s, got %s", parentTraceID, traceID)
	}

	// 检查 SpanID
	spanID := trace.SpanID(ctx)
	if spanID == "" {
		t.Error("Expected non-empty span ID")
	}

	// 检查 ParentSpanID
	psID := trace.ParentSpanID(ctx)
	if psID != parentSpanID {
		t.Errorf("Expected parent span ID %s, got %s", parentSpanID, psID)
	}
}

func TestNewSpan(t *testing.T) {
	ctx := context.Background()

	// 初始化追踪
	ctx = trace.InitTrace(ctx)

	// 创建新跨度
	newCtx, span := trace.NewSpan(ctx, "test-span")

	// 检查新上下文
	newTraceID := trace.TraceID(newCtx)
	if newTraceID == "" {
		t.Error("Expected non-empty trace ID")
	}

	newSpanID := trace.SpanID(newCtx)
	if newSpanID == "" {
		t.Error("Expected non-empty span ID")
	}

	// 检查跨度信息
	if span.Name != "test-span" {
		t.Errorf("Expected span name 'test-span', got '%s'", span.Name)
	}

	if span.TraceID != newTraceID {
		t.Errorf("Expected span trace ID %s, got %s", newTraceID, span.TraceID)
	}

	if span.SpanID != newSpanID {
		t.Errorf("Expected span ID %s, got %s", newSpanID, span.SpanID)
	}

	// 检查父跨度 ID
	if span.ParentSpanID == "" {
		t.Error("Expected non-empty parent span ID")
	}
}

func TestSpanEnd(t *testing.T) {
	ctx := context.Background()
	ctx = trace.InitTrace(ctx)

	// 创建跨度
	_, span := trace.NewSpan(ctx, "test-span")

	// 检查开始时间
	if span.StartTime.IsZero() {
		t.Error("Expected non-zero start time")
	}

	// 检查结束时间（应该为零）
	if !span.EndTime.IsZero() {
		t.Error("Expected zero end time before End()")
	}

	// 结束跨度
	span.End()

	// 检查结束时间
	if span.EndTime.IsZero() {
		t.Error("Expected non-zero end time after End()")
	}

	// 检查成功状态
	if !span.Success {
		t.Error("Expected success to be true")
	}
}

func TestSpanEndWithError(t *testing.T) {
	ctx := context.Background()
	ctx = trace.InitTrace(ctx)

	// 创建跨度
	_, span := trace.NewSpan(ctx, "test-span")

	// 结束跨度并记录错误
	err := testError{}
	span.EndWithError(err)

	// 检查错误
	if span.Error == nil {
		t.Error("Expected non-nil error")
	}

	// 检查成功状态
	if span.Success {
		t.Error("Expected success to be false")
	}
}

func TestSpanDuration(t *testing.T) {
	ctx := context.Background()
	ctx = trace.InitTrace(ctx)

	// 创建跨度
	_, span := trace.NewSpan(ctx, "test-span")

	// 等待一段时间
	time.Sleep(10 * time.Millisecond)

	// 结束跨度
	span.End()

	// 检查持续时间
	duration := span.Duration()
	if duration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", duration)
	}
}

func TestGetTraceInfo(t *testing.T) {
	ctx := context.Background()
	ctx = trace.InitTrace(ctx)

	// 获取追踪信息
	info := trace.GetTraceInfo(ctx)

	// 检查追踪信息
	if info["trace_id"] == "" {
		t.Error("Expected non-empty trace_id in info")
	}

	if info["span_id"] == "" {
		t.Error("Expected non-empty span_id in info")
	}

	if info["parent_span_id"] != "" {
		t.Errorf("Expected empty parent_span_id in info, got %s", info["parent_span_id"])
	}
}

func TestWithTraceID(t *testing.T) {
	ctx := context.Background()

	// 添加追踪 ID
	traceID := "test-trace-id"
	ctx = trace.WithTraceID(ctx, traceID)

	// 检查追踪 ID
	retrievedTraceID := trace.TraceID(ctx)
	if retrievedTraceID != traceID {
		t.Errorf("Expected trace ID %s, got %s", traceID, retrievedTraceID)
	}
}

func TestWithSpanID(t *testing.T) {
	ctx := context.Background()

	// 添加跨度 ID
	spanID := "test-span-id"
	ctx = trace.WithSpanID(ctx, spanID)

	// 检查跨度 ID
	retrievedSpanID := trace.SpanID(ctx)
	if retrievedSpanID != spanID {
		t.Errorf("Expected span ID %s, got %s", spanID, retrievedSpanID)
	}
}

func TestWithParentSpanID(t *testing.T) {
	ctx := context.Background()

	// 添加父跨度 ID
	parentSpanID := "test-parent-span-id"
	ctx = trace.WithParentSpanID(ctx, parentSpanID)

	// 检查父跨度 ID
	retrievedParentSpanID := trace.ParentSpanID(ctx)
	if retrievedParentSpanID != parentSpanID {
		t.Errorf("Expected parent span ID %s, got %s", parentSpanID, retrievedParentSpanID)
	}
}

func TestNewTraceID(t *testing.T) {
	// 生成新的追踪 ID
	traceID1 := trace.NewTraceID()
	traceID2 := trace.NewTraceID()

	// 检查追踪 ID 不为空
	if traceID1 == "" {
		t.Error("Expected non-empty trace ID")
	}

	if traceID2 == "" {
		t.Error("Expected non-empty trace ID")
	}

	// 检查追踪 ID 唯一性
	if traceID1 == traceID2 {
		t.Error("Expected unique trace IDs")
	}
}

func TestNewSpanID(t *testing.T) {
	// 生成新的跨度 ID
	spanID1 := trace.NewSpanID()
	spanID2 := trace.NewSpanID()

	// 检查跨度 ID 不为空
	if spanID1 == "" {
		t.Error("Expected non-empty span ID")
	}

	if spanID2 == "" {
		t.Error("Expected non-empty span ID")
	}

	// 检查跨度 ID 唯一性
	if spanID1 == spanID2 {
		t.Error("Expected unique span IDs")
	}
}

func TestSpanString(t *testing.T) {
	ctx := context.Background()
	ctx = trace.InitTrace(ctx)

	// 创建跨度
	_, span := trace.NewSpan(ctx, "test-span")
	span.End()

	// 检查字符串表示
	str := span.String()
	if str == "" {
		t.Error("Expected non-empty string representation")
	}

	// 检查字符串包含关键信息
	expectedSubstrings := []string{
		"Name: test-span",
		"TraceID:",
		"SpanID:",
		"Duration:",
		"Success: true",
	}

	for _, substr := range expectedSubstrings {
		if !contains(str, substr) {
			t.Errorf("Expected string to contain '%s', got '%s'", substr, str)
		}
	}
}

func TestNestedSpans(t *testing.T) {
	ctx := context.Background()

	// 初始化追踪
	ctx = trace.InitTrace(ctx)

	// 创建第一个跨度
	ctx1, span1 := trace.NewSpan(ctx, "span-1")

	// 创建第二个跨度（嵌套）
	_, span2 := trace.NewSpan(ctx1, "span-2")

	// 检查跨度关系
	if span2.ParentSpanID != span1.SpanID {
		t.Errorf("Expected span2 parent ID %s, got %s", span1.SpanID, span2.ParentSpanID)
	}

	if span2.TraceID != span1.TraceID {
		t.Errorf("Expected span2 trace ID %s, got %s", span1.TraceID, span2.TraceID)
	}

	// 结束跨度
	span2.End()
	span1.End()
}

func TestEmptyContext(t *testing.T) {
	ctx := context.Background()

	// 检查空上下文的追踪信息
	traceID := trace.TraceID(ctx)
	if traceID != "" {
		t.Errorf("Expected empty trace ID, got %s", traceID)
	}

	spanID := trace.SpanID(ctx)
	if spanID != "" {
		t.Errorf("Expected empty span ID, got %s", spanID)
	}

	parentSpanID := trace.ParentSpanID(ctx)
	if parentSpanID != "" {
		t.Errorf("Expected empty parent span ID, got %s", parentSpanID)
	}
}

type testError struct{}

func (e testError) Error() string {
	return "test error"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
