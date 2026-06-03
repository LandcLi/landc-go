package demo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/auth"
	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/LandcLi/landc-go/frame/pkg/middleware"
	"github.com/LandcLi/landc-go/frame/pkg/response"
	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	config.InitGlobalConfigWithConfig(&config.Config{
		Server: config.ServerConfig{Port: 8080},
	})
	auth.InitJWT(&auth.JWTConfig{
		Secret:     "test-secret",
		ExpireTime: 1 * time.Hour,
		Issuer:     "test",
	})

	RegisterDependencies()

	r := gin.New()
	r.Use(middleware.Trace())
	r.Use(middleware.Recovery())

	r.POST("/api/v1/login", Login)

	authorized := r.Group("/api/v1")
	authorized.Use(middleware.Auth())
	{
		authorized.POST("/user/create", CreateUser)
		authorized.GET("/user/get", GetUser)
		authorized.PUT("/user/update", UpdateUser)
		authorized.DELETE("/user/delete", DeleteUser)
		authorized.GET("/user/list", ListUsers)
	}

	return r
}

func getAdminToken() string {
	token, _ := auth.GenerateToken(1, "admin", "admin")
	return token
}

func TestLogin(t *testing.T) {
	r := setupTestRouter()

	body := `{"username":"admin","password":"admin123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != response.CodeSuccess {
		t.Errorf("Expected success code, got %d: %s", resp.Code, resp.Message)
	}
}

func TestLoginInvalidPassword(t *testing.T) {
	r := setupTestRouter()

	body := `{"username":"admin","password":"wrong"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code == response.CodeSuccess {
		t.Error("Login should fail with wrong password")
	}
}

func TestCreateUser(t *testing.T) {
	r := setupTestRouter()
	token := getAdminToken()

	body := `{"username":"newuser","password":"123456","email":"new@test.com"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/user/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != response.CodeSuccess {
		t.Errorf("Expected success, got code %d: %s", resp.Code, resp.Message)
	}
}

func TestGetUser(t *testing.T) {
	r := setupTestRouter()
	token := getAdminToken()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/user/get?id=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != response.CodeSuccess {
		t.Errorf("Expected success, got code %d: %s", resp.Code, resp.Message)
	}
}

func TestListUsers(t *testing.T) {
	r := setupTestRouter()
	token := getAdminToken()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/user/list?page=1&page_size=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != response.CodeSuccess {
		t.Errorf("Expected success, got code %d: %s", resp.Code, resp.Message)
	}
}

func TestUnauthorizedAccess(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/user/get?id=1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestDeleteUser(t *testing.T) {
	r := setupTestRouter()
	token := getAdminToken()

	// 先创建一个用户
	body := `{"username":"toDelete","password":"123456","email":"del@test.com"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/user/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	// 删除用户 ID=2
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("DELETE", "/api/v1/user/delete?id=2", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	var resp response.Response
	json.Unmarshal(w2.Body.Bytes(), &resp)

	if resp.Code != response.CodeSuccess {
		t.Errorf("Expected success, got code %d: %s", resp.Code, resp.Message)
	}
}
