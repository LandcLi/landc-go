package tests

import (
	"net/http"
	"testing"

	"github.com/LandcLi/landc-go/api/trace"
)

func TestNewTrace(t *testing.T) {
	// 测试创建新的跟踪上下文
	traceObj := trace.NewTrace()
	if traceObj == nil {
		t.Errorf("NewTrace should not return nil")
	}
	if traceObj.TraceID == "" {
		t.Errorf("TraceID should not be empty")
	}
	if len(traceObj.TraceID) != trace.TraceIDLength {
		t.Errorf("TraceID length should be %d, got %d", trace.TraceIDLength, len(traceObj.TraceID))
	}
}

func TestExtractFromRequest(t *testing.T) {
	// 测试从没有TraceID的请求中提取
	req, err := http.NewRequest("GET", "/api/hello", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	traceObj := trace.ExtractFromRequest(req)
	if traceObj == nil {
		t.Errorf("ExtractFromRequest should not return nil")
	}
	if traceObj.TraceID == "" {
		t.Errorf("TraceID should not be empty")
	}

	// 测试从有TraceID的请求中提取
	testTraceID := "5d41402abc4b2a76b9719d911017c592"
	reqWithTrace := req.Clone(req.Context())
	reqWithTrace.Header.Set(trace.TraceIDHeader, testTraceID)

	traceObjWithTrace := trace.ExtractFromRequest(reqWithTrace)
	if traceObjWithTrace == nil {
		t.Errorf("ExtractFromRequest should not return nil for request with TraceID")
	}
	if traceObjWithTrace.TraceID != testTraceID {
		t.Errorf("Expected TraceID '%s', got '%s'", testTraceID, traceObjWithTrace.TraceID)
	}
}

func TestInjectToRequest(t *testing.T) {
	traceObj := trace.NewTrace()
	req, err := http.NewRequest("GET", "/api/hello", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// 测试注入TraceID到请求
	traceObj.InjectToRequest(req)
	if req.Header.Get(trace.TraceIDHeader) != traceObj.TraceID {
		t.Errorf("Expected TraceID '%s' in request header, got '%s'", traceObj.TraceID, req.Header.Get(trace.TraceIDHeader))
	}
}

func TestInjectToHeaders(t *testing.T) {
	traceObj := trace.NewTrace()
	headers := make(map[string]string)

	// 测试注入TraceID到 headers
	traceObj.InjectToHeaders(headers)
	if headers[trace.TraceIDHeader] != traceObj.TraceID {
		t.Errorf("Expected TraceID '%s' in headers, got '%s'", traceObj.TraceID, headers[trace.TraceIDHeader])
	}
}

func TestGenerateTraceID(t *testing.T) {
	// 测试生成TraceID
	traceID := trace.GenerateTraceID()
	if traceID == "" {
		t.Errorf("GenerateTraceID should not return empty string")
	}
	if len(traceID) != trace.TraceIDLength {
		t.Errorf("TraceID length should be %d, got %d", trace.TraceIDLength, len(traceID))
	}

	// 测试生成多个不同的TraceID
	traceID1 := trace.GenerateTraceID()
	traceID2 := trace.GenerateTraceID()
	if traceID1 == traceID2 {
		t.Errorf("GenerateTraceID should return different values each time")
	}
}

func TestValidateTraceID(t *testing.T) {
	// 测试验证有效的TraceID
	validTraceID := "5d41402abc4b2a76b9719d911017c592"
	if !trace.ValidateTraceID(validTraceID) {
		t.Errorf("ValidateTraceID should return true for valid TraceID '%s'", validTraceID)
	}

	// 测试验证无效的TraceID（长度错误）
	invalidLengthTraceID := "5d41402abc4b2a76b9719d911017c5"
	if trace.ValidateTraceID(invalidLengthTraceID) {
		t.Errorf("ValidateTraceID should return false for TraceID with invalid length")
	}

	// 测试验证无效的TraceID（格式错误）
	invalidFormatTraceID := "invalid-trace-id"
	if trace.ValidateTraceID(invalidFormatTraceID) {
		t.Errorf("ValidateTraceID should return false for TraceID with invalid format")
	}
}

func TestExtractTraceFromHeaders(t *testing.T) {
	// 测试从没有TraceID的headers中提取
	headers := make(map[string]string)
	traceObj := trace.ExtractTraceFromHeaders(headers)
	if traceObj == nil {
		t.Errorf("ExtractTraceFromHeaders should not return nil")
	}
	if traceObj.TraceID == "" {
		t.Errorf("TraceID should not be empty")
	}

	// 测试从有TraceID的headers中提取
	testTraceID := "5d41402abc4b2a76b9719d911017c592"
	headersWithTrace := make(map[string]string)
	headersWithTrace[trace.TraceIDHeader] = testTraceID

	traceObjWithTrace := trace.ExtractTraceFromHeaders(headersWithTrace)
	if traceObjWithTrace == nil {
		t.Errorf("ExtractTraceFromHeaders should not return nil for headers with TraceID")
	}
	if traceObjWithTrace.TraceID != testTraceID {
		t.Errorf("Expected TraceID '%s', got '%s'", testTraceID, traceObjWithTrace.TraceID)
	}
}
