package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LandcLi/landc-go/mvc/pkg/meta"
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
	ctx.SetHeader("X-Response-Header", "response-value")

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
		req, _ := http.NewRequest("GET", "/test/123?name=test", nil)
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
		req, _ := http.NewRequest("GET", "/testnoparams", nil)
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

		req, _ := http.NewRequest("GET", "/api/v1/item/999?name=GroupTest", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		expectedBody := `{"id":999,"name":"GroupTest"}`
		if w.Body.String() != expectedBody {
			t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
		}
	})

	t.Run("Test context parameter", func(t *testing.T) {
		contextController := &ContextController{}

		err := server.RegisterHandler(contextController)
		if err != nil {
			t.Fatalf("Failed to register context handler: %v", err)
		}

		req, _ := http.NewRequest("GET", "/context/123", nil)
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

		expectedBody := `{"id":123,"received":"custom-value"}`
		if w.Body.String() != expectedBody {
			t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
		}
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
		req, _ := http.NewRequest("GET", "/test/456?name=John", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		expectedBody := `{"id":456,"name":"John"}`
		if w.Body.String() != expectedBody {
			t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
		}
	})

	t.Run("Parse query parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test/789?name=Alice", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		expectedBody := `{"id":789,"name":"Alice"}`
		if w.Body.String() != expectedBody {
			t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
		}
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
		config := &ServerConfig{
			Addr: ":9090",
		}
		server := NewServer(config)

		if server.config.Addr != ":9090" {
			t.Errorf("Expected custom addr :9090, got %s", server.config.Addr)
		}
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
