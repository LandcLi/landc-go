package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/auth"
	"github.com/LandcLi/landc-go/frame/pkg/response"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestTraceMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(Trace())
	r.GET("/test", func(c *gin.Context) {
		traceID := c.GetString("trace_id")
		c.JSON(200, gin.H{"trace_id": traceID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["trace_id"] == "" {
		t.Error("trace_id should not be empty")
	}

	// 验证响应头也有 X-Trace-ID
	if w.Header().Get("X-Trace-ID") == "" {
		t.Error("X-Trace-ID response header should not be empty")
	}
}

func TestTraceMiddlewareWithHeader(t *testing.T) {
	r := gin.New()
	r.Use(Trace())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"trace_id": c.GetString("trace_id")})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("X-Trace-ID", "custom-trace-id")
	r.ServeHTTP(w, req)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["trace_id"] != "custom-trace-id" {
		t.Errorf("Expected 'custom-trace-id', got '%s'", resp["trace_id"])
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(Trace())
	r.Use(Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != response.CodeInternalServerError {
		t.Errorf("Expected code %d, got %d", response.CodeInternalServerError, resp.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	// 普通请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected CORS header")
	}

	// OPTIONS 预检请求
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("OPTIONS", "/test", http.NoBody)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNoContent {
		t.Errorf("Expected 204 for OPTIONS, got %d", w2.Code)
	}
}

func TestAuthMiddleware(t *testing.T) {
	auth.InitJWT(&auth.JWTConfig{
		Secret:     "test-secret-key-0123456789abcdef0123456789abcdef",
		ExpireTime: 1 * time.Hour,
		Issuer:     "test",
	})

	r := gin.New()
	r.Use(Auth())
	r.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(200, gin.H{"user_id": userID})
	})

	// 无 token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", http.NoBody)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 without token, got %d", w.Code)
	}

	// 无效 token
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/protected", http.NoBody)
	req2.Header.Set("Authorization", "Bearer invalid-token")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for invalid token, got %d", w2.Code)
	}

	// 有效 token
	token, _ := auth.GenerateToken(42, "testuser", "admin")
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/protected", http.NoBody)
	req3.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("Expected 200 for valid token, got %d", w3.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &resp)
	if resp["user_id"].(float64) != 42 {
		t.Errorf("Expected user_id 42, got %v", resp["user_id"])
	}
}

func TestRoleRequiredMiddleware(t *testing.T) {
	auth.InitJWT(&auth.JWTConfig{
		Secret:     "test-secret-key-0123456789abcdef0123456789abcdef",
		ExpireTime: 1 * time.Hour,
		Issuer:     "test",
	})

	r := gin.New()
	r.Use(Auth())
	r.Use(RoleRequired("admin"))
	r.GET("/admin", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	// admin 角色
	adminToken, _ := auth.GenerateToken(1, "admin", "admin")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("Expected 200 for admin, got %d", w.Code)
	}

	// user 角色
	userToken, _ := auth.GenerateToken(2, "user", "user")
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/admin", http.NoBody)
	req2.Header.Set("Authorization", "Bearer "+userToken)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for user role, got %d", w2.Code)
	}
}
