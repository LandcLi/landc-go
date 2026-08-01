package tests

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LandcLi/landc-go/api/core"
	ginApi "github.com/LandcLi/landc-go/api/middleware/gin"
	"github.com/gin-gonic/gin"
)

// ========== Header Tests ==========

func TestHeaderExtractor(t *testing.T) {
	extractor := core.NewDefaultHeaderExtractor()

	req, err := http.NewRequest("GET", "/api/hello", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.RemoteAddr = "192.168.1.1:12345"

	headers := extractor.ExtractHeaders(req)
	if headers == nil {
		t.Fatal("ExtractHeaders should not return nil")
	}

	if headers["X-Request-ID"] == "" {
		t.Error("X-Request-ID should not be empty")
	}

	if headers["X-Client-IP"] == "" {
		t.Error("X-Client-IP should not be empty")
	}

	// 自定义 User-Agent
	reqWithUA := req.Clone(req.Context())
	reqWithUA.Header.Set("User-Agent", "Mozilla/5.0")
	headersWithUA := extractor.ExtractHeaders(reqWithUA)
	if headersWithUA["X-User-Agent"] != "Mozilla/5.0" {
		t.Errorf("Expected User-Agent 'Mozilla/5.0', got '%s'", headersWithUA["X-User-Agent"])
	}

	// 自定义提取器
	customExtractor := core.NewDefaultHeaderExtractor().WithCustomExtractor("X-Custom-Header", func(r *http.Request) string {
		return r.Header.Get("X-Custom-Header")
	})
	reqWithCustom := req.Clone(req.Context())
	reqWithCustom.Header.Set("X-Custom-Header", "custom-value")
	customHeaders := customExtractor.ExtractHeaders(reqWithCustom)
	if customHeaders["X-Custom-Header"] != "custom-value" {
		t.Errorf("Expected 'custom-value', got '%s'", customHeaders["X-Custom-Header"])
	}
}

func TestClientIPTrustedProxy(t *testing.T) {
	extractor := core.NewDefaultHeaderExtractor()

	// 来自可信代理的请求，应信任 X-Forwarded-For
	req, _ := http.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345" // 可信代理（私有网络）
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 192.168.1.1")

	headers := extractor.ExtractHeaders(req)
	if headers["X-Client-IP"] != "203.0.113.50" {
		t.Errorf("Expected client IP '203.0.113.50', got '%s'", headers["X-Client-IP"])
	}

	// 来自非可信代理的请求，不应信任 X-Forwarded-For
	req2, _ := http.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "203.0.113.100:12345" // 非可信代理
	req2.Header.Set("X-Forwarded-For", "1.2.3.4")

	headers2 := extractor.ExtractHeaders(req2)
	if headers2["X-Client-IP"] != "203.0.113.100" {
		t.Errorf("Expected client IP '203.0.113.100' (direct), got '%s'", headers2["X-Client-IP"])
	}
}

func TestClientIPXRealIP(t *testing.T) {
	extractor := core.NewDefaultHeaderExtractor()

	req, _ := http.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:8080" // 可信代理
	req.Header.Set("X-Real-IP", "203.0.113.77")

	headers := extractor.ExtractHeaders(req)
	if headers["X-Client-IP"] != "203.0.113.77" {
		t.Errorf("Expected '203.0.113.77', got '%s'", headers["X-Client-IP"])
	}
}

func TestHeaderProcessor(t *testing.T) {
	processor := core.NewHeaderProcessor()

	req, err := http.NewRequest("GET", "/api/hello", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rw := &mockResponseWriter{}
	headers := processor.Process(req, rw)
	if headers == nil {
		t.Fatal("Process should not return nil")
	}

	// 自定义提取器
	customProcessor := core.NewHeaderProcessor().WithCustomExtractor("X-Test-Header", func(r *http.Request) string {
		return "test-value"
	})
	customHeaders := customProcessor.Process(req, rw)
	if customHeaders["X-Test-Header"] != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", customHeaders["X-Test-Header"])
	}
}

func TestTrustedProxiesConfig(t *testing.T) {
	processor := core.NewHeaderProcessor().WithTrustedProxies([]string{"10.0.0.0/8"})

	req, _ := http.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.5:8080"
	req.Header.Set("X-Forwarded-For", "8.8.8.8")

	rw := &mockResponseWriter{}
	headers := processor.Process(req, rw)

	if headers["X-Client-IP"] != "8.8.8.8" {
		t.Errorf("Expected '8.8.8.8', got '%s'", headers["X-Client-IP"])
	}
}

// ========== Gin Error Converter Tests ==========

func TestGinErrorConverter(t *testing.T) {
	converter := ginApi.NewGinErrorConverter()

	// core.Error 直接返回
	apiErr := core.NewError(core.ErrorCodeBadRequest, "Bad Request")
	convertedErr := converter.ConvertError(apiErr)
	if convertedErr != apiErr {
		t.Error("ConvertError should return the same core.Error instance")
	}

	// 普通 error
	stdErr := errors.New("some error")
	convertedStdErr := converter.ConvertError(stdErr)
	if convertedStdErr == nil {
		t.Fatal("ConvertError should not return nil")
	}
	if convertedStdErr.Code != core.ErrorCodeInternalServerError {
		t.Errorf("Expected code %d, got %d", core.ErrorCodeInternalServerError, convertedStdErr.Code)
	}

	// 自定义转换器
	customConverter := ginApi.NewGinErrorConverter().WithCustomConverter(func(err error) *core.Error {
		return core.NewError(core.ErrorCodeBusinessError, "Custom Business Error")
	})
	customErr := customConverter.ConvertError(stdErr)
	if customErr.Code != core.ErrorCodeBusinessError {
		t.Errorf("Expected code %d, got %d", core.ErrorCodeBusinessError, customErr.Code)
	}

	// Gin ErrorTypeBind
	bindErr := gin.Error{Type: gin.ErrorTypeBind, Err: errors.New("bind error")}
	convertedBindErr := converter.ConvertGinError(&bindErr)
	if convertedBindErr.Code != core.ErrorCodeValidationFailed {
		t.Errorf("Expected code %d, got %d", core.ErrorCodeValidationFailed, convertedBindErr.Code)
	}

	// Gin ErrorTypePublic
	publicErr := gin.Error{Type: gin.ErrorTypePublic, Err: errors.New("public error")}
	convertedPublicErr := converter.ConvertGinError(&publicErr)
	if convertedPublicErr.Code != core.ErrorCodeBadRequest {
		t.Errorf("Expected code %d, got %d", core.ErrorCodeBadRequest, convertedPublicErr.Code)
	}

	// Gin ErrorTypePrivate
	privateErr := gin.Error{Type: gin.ErrorTypePrivate, Err: errors.New("private error")}
	convertedPrivateErr := converter.ConvertGinError(&privateErr)
	if convertedPrivateErr.Code != core.ErrorCodeInternalServerError {
		t.Errorf("Expected code %d, got %d", core.ErrorCodeInternalServerError, convertedPrivateErr.Code)
	}

	// Gin ErrorTypeRender
	renderErr := gin.Error{Type: gin.ErrorTypeRender, Err: errors.New("render error")}
	convertedRenderErr := converter.ConvertGinError(&renderErr)
	if convertedRenderErr.Code != core.ErrorCodeInternalServerError {
		t.Errorf("Expected code %d, got %d", core.ErrorCodeInternalServerError, convertedRenderErr.Code)
	}

	// nil error
	if converter.ConvertGinError(nil) != nil {
		t.Error("ConvertGinError(nil) should return nil")
	}
	if converter.ConvertError(nil) != nil {
		t.Error("ConvertError(nil) should return nil")
	}
}

// ========== Gin Middleware Integration Tests ==========

func TestGinMiddlewareBasicSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ginApi.GinMiddleware())

	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"hello": "world"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp core.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Code != int(core.CodeSuccess) {
		t.Errorf("Expected code %d, got %d", core.CodeSuccess, resp.Code)
	}
	if resp.TraceID == "" {
		t.Error("TraceID should not be empty")
	}
	if resp.Latency < 0 {
		t.Errorf("Latency should be >= 0, got %d", resp.Latency)
	}
}

func TestGinMiddlewareWithError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ginApi.GinMiddleware())

	r.GET("/error", func(c *gin.Context) {
		c.Error(core.NewError(core.ErrorCodeBadRequest, "invalid param"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)
	r.ServeHTTP(w, req)

	var resp core.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Code != int(core.ErrorCodeBadRequest) {
		t.Errorf("Expected code %d, got %d", core.ErrorCodeBadRequest, resp.Code)
	}
	if resp.Message != "invalid param" {
		t.Errorf("Expected message 'invalid param', got '%s'", resp.Message)
	}
}

func TestGinMiddlewareWithTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ginApi.GinMiddleware())

	r.GET("/trace", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/trace", nil)
	req.Header.Set("X-Trace-ID", "my-custom-trace-id")
	r.ServeHTTP(w, req)

	var resp core.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.TraceID != "my-custom-trace-id" {
		t.Errorf("Expected trace_id 'my-custom-trace-id', got '%s'", resp.TraceID)
	}
}

func TestGinMiddlewareNoBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ginApi.GinMiddleware())

	r.GET("/empty", func(c *gin.Context) {
		// handler 不写任何响应
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/empty", nil)
	r.ServeHTTP(w, req)

	var resp core.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Code != int(core.CodeSuccess) {
		t.Errorf("Expected success code, got %d", resp.Code)
	}
}

func TestGinMiddlewareResponseWrittenToClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ginApi.GinMiddleware())

	r.GET("/check", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "test"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/check", nil)
	r.ServeHTTP(w, req)

	// 关键：验证响应体不为空（修复 #1 的验证）
	if w.Body.Len() == 0 {
		t.Fatal("Response body should not be empty - response was not written to client!")
	}

	// Content-Type 应为 JSON
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", ct)
	}
}

func TestGinMiddlewareCustomErrorParser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	config := ginApi.NewGinMiddlewareConfig().WithCustomErrorParser(func(c *gin.Context, body []byte) error {
		// 模拟自定义错误解析：如果body包含"error"则返回错误
		if len(body) > 0 {
			var data map[string]interface{}
			if json.Unmarshal(body, &data) == nil {
				if _, ok := data["error"]; ok {
					return core.NewError(core.ErrorCodeBusinessError, "custom parsed error")
				}
			}
		}
		return nil
	})
	r.Use(ginApi.GinMiddlewareWithConfig(config))

	r.GET("/custom-error", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"error": "something wrong"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/custom-error", nil)
	r.ServeHTTP(w, req)

	var resp core.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != int(core.ErrorCodeBusinessError) {
		t.Errorf("Expected code %d, got %d", core.ErrorCodeBusinessError, resp.Code)
	}
}

// ========== Mock ==========

type mockResponseWriter struct {
	headers http.Header
}

func (m *mockResponseWriter) Header() http.Header {
	if m.headers == nil {
		m.headers = make(http.Header)
	}
	return m.headers
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {}
