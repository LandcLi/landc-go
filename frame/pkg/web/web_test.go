package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/LandcLi/landc-go/frame/pkg/meta"
	"github.com/LandcLi/landc-go/frame/pkg/trace"
	"github.com/gin-gonic/gin"
)

type (
	TestController struct{}

	TestRequest struct {
		meta.Meta `path:"/test/:id" method:"GET"`
		ID        int64  `source:"path" name:"id"`
		Name      string `source:"query" name:"name"`
	}

	TestResponse struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	GroupController struct {
		meta.Meta `path:"/api/v1"`
	}

	GroupRequest struct {
		meta.Meta `path:"/item/:id" method:"GET"`
		ID        int64  `source:"path" name:"id"`
		Name      string `source:"query" name:"name"`
	}

	GroupResponse struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	ContextController struct{}

	ContextRequest struct {
		meta.Meta `path:"/context/:id" method:"GET"`
		ID        int64 `source:"path" name:"id"`
	}

	ContextResponse struct {
		ID       int64  `json:"id"`
		Received string `json:"received"`
	}
)

func (c *TestController) Test(req *TestRequest) (*TestResponse, error) {
	return &TestResponse{
		ID:   req.ID,
		Name: req.Name,
	}, nil
}

func (c *TestController) TestNoParams() (map[string]string, error) {
	return map[string]string{"message": "no params"}, nil
}

func (c *GroupController) GetItem(req *GroupRequest) (*GroupResponse, error) {
	return &GroupResponse{
		ID:   req.ID,
		Name: req.Name,
	}, nil
}

func (c *ContextController) TestContext(ctx *LandcContext, req *ContextRequest) (*ContextResponse, error) {
	customHeader := ctx.GetHeader("X-Custom-Header")
	ctx.Header("X-Response-Header", "response-value")

	return &ContextResponse{
		ID:       req.ID,
		Received: customHeader,
	}, nil
}

func TestRegisterHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := NewServer(&ServerConfig{
		Addr: ":8080",
	})

	controller := &TestController{}

	err := server.RegisterHandler(controller)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	router := server.Engine()

	t.Run("Test with path and query params", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test/123?name=test", http.NoBody)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
			t.Errorf("Expected JSON content type, got %s", w.Header().Get("Content-Type"))
		}
	})

	t.Run("Test no params", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/testnoparams", http.NoBody)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Test group registration", func(t *testing.T) {
		groupController := &GroupController{}

		err := server.RegisterHandler(groupController)
		if err != nil {
			t.Fatalf("Failed to register group handler: %v", err)
		}

		req, _ := http.NewRequest("GET", "/api/v1/item/999?name=GroupTest", http.NoBody)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Response is wrapped in standard response format
		expectedBody := `{"code":10000,"data":{"id":999,"name":"GroupTest"},"message":"success"}`
		assertJSONBody(t, w.Body.String(), expectedBody)
	})

	t.Run("Test context parameter", func(t *testing.T) {
		contextController := &ContextController{}

		err := server.RegisterHandler(contextController)
		if err != nil {
			t.Fatalf("Failed to register context handler: %v", err)
		}

		req, _ := http.NewRequest("GET", "/context/123", http.NoBody)
		req.Header.Set("X-Custom-Header", "custom-value")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		responseHeader := w.Header().Get("X-Response-Header")
		if responseHeader != "response-value" {
			t.Errorf("Expected response header 'response-value', got %s", responseHeader)
		}

		// Response is wrapped in standard response format
		expectedBody := `{"code":10000,"data":{"id":123,"received":"custom-value"},"message":"success"}`
		assertJSONBody(t, w.Body.String(), expectedBody)
	})
}

func TestParamParsing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := NewServer(&ServerConfig{
		Addr: ":8080",
	})

	controller := &TestController{}

	err := server.RegisterHandler(controller)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	router := server.Engine()

	t.Run("Parse path parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test/456?name=John", http.NoBody)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Response is wrapped in standard response format
		expectedBody := `{"code":10000,"data":{"id":456,"name":"John"},"message":"success"}`
		assertJSONBody(t, w.Body.String(), expectedBody)
	})

	t.Run("Parse query parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test/789?name=Alice", http.NoBody)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Response is wrapped in standard response format
		expectedBody := `{"code":10000,"data":{"id":789,"name":"Alice"},"message":"success"}`
		assertJSONBody(t, w.Body.String(), expectedBody)
	})
}

func TestServerConfig(t *testing.T) {
	t.Run("Default config", func(t *testing.T) {
		server := NewServer(nil)

		if server.config.Addr != ":8080" {
			t.Errorf("Expected default addr :8080, got %s", server.config.Addr)
		}
	})

	t.Run("Custom config", func(t *testing.T) {
		sc := &ServerConfig{
			Addr: ":9090",
		}
		server := NewServer(sc)

		if server.config.Addr != ":9090" {
			t.Errorf("Expected custom addr :9090, got %s", server.config.Addr)
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("New returns valid server", func(t *testing.T) {
		server := New()
		if server == nil {
			t.Fatal("Expected non-nil server")
		}
		if server.config.Addr != ":8080" {
			t.Errorf("Expected default addr :8080, got %s", server.config.Addr)
		}
		if server.config.ReadTimeout != 60*time.Second {
			t.Errorf("Expected default ReadTimeout 60s, got %v", server.config.ReadTimeout)
		}
	})

	t.Run("New reads from global config", func(t *testing.T) {
		// 设置全局配置
		cfg := config.DefaultConfig()
		cfg.Server.Addr = "0.0.0.0"
		cfg.Server.Port = 3000
		cfg.Server.ReadTimeout = 30
		cfg.Server.WriteTimeout = 15
		config.InitGlobalConfigWithConfig(cfg)

		server := New()

		if server.config.Addr != "0.0.0.0:3000" {
			t.Errorf("Expected addr 0.0.0.0:3000, got %s", server.config.Addr)
		}
		if server.config.ReadTimeout != 30*time.Second {
			t.Errorf("Expected ReadTimeout 30s, got %v", server.config.ReadTimeout)
		}
		if server.config.WriteTimeout != 15*time.Second {
			t.Errorf("Expected WriteTimeout 15s, got %v", server.config.WriteTimeout)
		}

		// 清理，避免影响其他测试
		config.InitGlobalConfigWithConfig(config.DefaultConfig())
	})
}

func TestEngine(t *testing.T) {
	server := NewServer(nil)

	engine := server.Engine()

	if engine == nil {
		t.Error("Expected non-nil engine")
	}
}

func TestIsExported(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Exported", "Exported", true},
		{"exported", "exported", false},
		{"", "", false},
		{"A", "A", true},
		{"a", "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isExported(tt.input)
			if result != tt.expected {
				t.Errorf("isExported(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
// assertJSONBody 断言响应体与期望的 JSON 结构一致，忽略 trace_id。
//
// trace_id 是每个请求随机生成的，createHandler 会为所有响应补上它。
// 对响应「结构」的严格比对不应该因为这一个随机字段而失败，
// 但其余字段仍需精确匹配（防止响应结构被无意改动）。
func assertJSONBody(t *testing.T, got, want string) {
	t.Helper()
	var gotMap, wantMap map[string]interface{}
	if err := json.Unmarshal([]byte(got), &gotMap); err != nil {
		t.Fatalf("响应不是合法 JSON: %s", got)
	}
	if err := json.Unmarshal([]byte(want), &wantMap); err != nil {
		t.Fatalf("期望值不是合法 JSON: %s", want)
	}
	delete(gotMap, "trace_id")
	if !reflect.DeepEqual(gotMap, wantMap) {
		t.Errorf("响应不符\n got: %s\nwant: %s", got, want)
	}
}

// TestEnsureTraceIDPropagatesUpstream 没有 trace 中间件时，必须沿用上游传来的 X-Trace-ID。
//
// 分布式追踪的核心语义：网关/调用方注入的 ID 要一路带下去。
// 如果这里改成自己生成一个，同一个请求在各服务里各有各的 ID，日志永远串不起来。
func TestEnsureTraceIDPropagatesUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/x", nil)
	c.Request.Header.Set("X-Trace-ID", "upstream-abc")

	got := ensureTraceID(c)
	if got != "upstream-abc" {
		t.Errorf("trace_id = %q, want upstream-abc（未沿用上游）", got)
	}
	if ctxTrace := trace.TraceID(c.Request.Context()); ctxTrace != "upstream-abc" {
		t.Errorf("request context 中的 trace_id = %q, want upstream-abc", ctxTrace)
	}
	if h := w.Header().Get("X-Trace-ID"); h != "upstream-abc" {
		t.Errorf("响应头 X-Trace-ID = %q, want upstream-abc", h)
	}
}

// TestEnsureTraceIDGeneratesWhenAbsent 上游没带时必须生成一个，且写进 context 与响应头。
func TestEnsureTraceIDGeneratesWhenAbsent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/x", nil)

	got := ensureTraceID(c)
	if got == "" {
		t.Fatal("trace_id 不应为空")
	}
	if trace.TraceID(c.Request.Context()) == "" {
		t.Error("trace_id 未写入 request context，下游 ProxyClient 取不到")
	}
	if w.Header().Get("X-Trace-ID") == "" {
		t.Error("trace_id 未写入响应头")
	}
}
