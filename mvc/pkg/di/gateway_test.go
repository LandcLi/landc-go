package di

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// ============ Test Types ============

type TestLoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type TestLoginResp struct {
	Token  string `json:"token"`
	UserID uint   `json:"user_id"`
}

type TestGetUserReq struct {
	UserID uint `json:"user_id" binding:"required"`
}

type TestGetUserResp struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// TestUserController is the controller interface
type TestUserController interface {
	Login(ctx context.Context, req *TestLoginReq) (*TestLoginResp, error)
	GetUser(ctx context.Context, req *TestGetUserReq) (*TestGetUserResp, error)
}

// localTestUserController is the local implementation
type localTestUserController struct{}

func (c *localTestUserController) Login(ctx context.Context, req *TestLoginReq) (*TestLoginResp, error) {
	if req.Username == "admin" && req.Password == "123456" {
		return &TestLoginResp{Token: "test-token-123", UserID: 1}, nil
	}
	return nil, fmt.Errorf("invalid credentials")
}

func (c *localTestUserController) GetUser(ctx context.Context, req *TestGetUserReq) (*TestGetUserResp, error) {
	if req.UserID == 1 {
		return &TestGetUserResp{Username: "admin", Email: "admin@test.com"}, nil
	}
	return nil, fmt.Errorf("user not found")
}

// Test routes
var testUserRoutes = Routes{
	"Login":   {Method: "POST", Path: "/api/user/login"},
	"GetUser": {Method: "POST", Path: "/api/user/get-info"},
}

// ============ Tests ============

func TestGateway_LocalMode(t *testing.T) {
	// Clean up
	globalContainer = NewContainer()

	gin.SetMode(gin.TestMode)

	// Create gateway and provide local implementation
	gw := NewGateway[TestUserController]("test.user.controller", testUserRoutes)
	gw.Provide(&localTestUserController{})

	// Verify we can get the implementation
	ctrl := gw.Get()
	resp, err := ctrl.Login(context.Background(), &TestLoginReq{Username: "admin", Password: "123456"})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if resp.Token != "test-token-123" {
		t.Errorf("Expected token 'test-token-123', got '%s'", resp.Token)
	}
}

func TestGateway_RegisterRoutes(t *testing.T) {
	// Clean up
	globalContainer = NewContainer()

	gin.SetMode(gin.TestMode)

	// Create gateway and provide local implementation
	gw := NewGateway[TestUserController]("test.user.controller2", testUserRoutes)
	gw.Provide(&localTestUserController{})

	// Register routes to Gin
	r := gin.New()
	gw.RegisterRoutes(r)

	// Test Login endpoint
	body := `{"username":"admin","password":"123456"}`
	req := httptest.NewRequest("POST", "/api/user/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var loginResp TestLoginResp
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if loginResp.Token != "test-token-123" {
		t.Errorf("Expected token 'test-token-123', got '%s'", loginResp.Token)
	}
	if loginResp.UserID != 1 {
		t.Errorf("Expected user_id 1, got %d", loginResp.UserID)
	}

	// Test error case
	body = `{"username":"wrong","password":"wrong"}`
	req = httptest.NewRequest("POST", "/api/user/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestGateway_RemoteMode(t *testing.T) {
	// Clean up
	globalContainer = NewContainer()

	gin.SetMode(gin.TestMode)

	// Step 1: Set up a "remote" server with local implementation
	gw := NewGateway[TestUserController]("test.user.controller3", testUserRoutes)
	gw.Provide(&localTestUserController{})

	r := gin.New()
	gw.RegisterRoutes(r)

	// Start test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Step 2: Create a remote proxy client
	client := NewProxyClient(server.URL, testUserRoutes)

	// Step 3: Use the proxy client to call remote service
	loginResp, err := Call[TestLoginResp](client, context.Background(), "Login", &TestLoginReq{
		Username: "admin",
		Password: "123456",
	})
	if err != nil {
		t.Fatalf("Remote login failed: %v", err)
	}
	if loginResp.Token != "test-token-123" {
		t.Errorf("Expected token 'test-token-123', got '%s'", loginResp.Token)
	}

	// Test GetUser
	getUserResp, err := Call[TestGetUserResp](client, context.Background(), "GetUser", &TestGetUserReq{
		UserID: 1,
	})
	if err != nil {
		t.Fatalf("Remote GetUser failed: %v", err)
	}
	if getUserResp.Username != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", getUserResp.Username)
	}

	// Test error case
	_, err = Call[TestLoginResp](client, context.Background(), "Login", &TestLoginReq{
		Username: "wrong",
		Password: "wrong",
	})
	if err == nil {
		t.Error("Expected error for invalid credentials")
	}
}

func TestProxyClient_Integration(t *testing.T) {
	// This test demonstrates the full pattern:
	// 1. Server side: Gateway with local impl + RegisterRoutes
	// 2. Client side: ProxyClient + Call[T] to build SDK

	// Clean up
	globalContainer = NewContainer()

	gin.SetMode(gin.TestMode)

	// === Server Side ===
	serverGw := NewGateway[TestUserController]("server.user", testUserRoutes)
	serverGw.Provide(&localTestUserController{})

	r := gin.New()
	serverGw.RegisterRoutes(r)
	server := httptest.NewServer(r)
	defer server.Close()

	// === Client Side (simulating what an SDK would look like) ===
	// With the framework, SDK code is reduced to:
	type remoteUserCtrl struct {
		client *ProxyClient
	}

	newRemoteCtrl := func(baseURL string) *remoteUserCtrl {
		return &remoteUserCtrl{
			client: NewProxyClient(baseURL, testUserRoutes),
		}
	}

	// Each method is just one line:
	remote := newRemoteCtrl(server.URL)

	resp, err := Call[TestLoginResp](remote.client, context.Background(), "Login", &TestLoginReq{
		Username: "admin",
		Password: "123456",
	})
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	if resp.Token != "test-token-123" {
		t.Errorf("Expected 'test-token-123', got '%s'", resp.Token)
	}
}

// ============ Module Tests ============

func TestModule_Basic(t *testing.T) {
	globalContainer = NewContainer()
	globalModuleManager = &ModuleManager{
		modules: make(map[string]*Module),
		order:   make([]string, 0),
	}

	ctx := context.Background()

	userModule := NewModule("user-manager").
		OnInit(func(ctx context.Context) error {
			Provide[TestUserController]("mod.user.ctrl", &localTestUserController{})
			return nil
		})

	err := UseModule(ctx, userModule)
	if err != nil {
		t.Fatalf("Failed to use module: %v", err)
	}

	ctrl := Require[TestUserController]("mod.user.ctrl")
	resp, _ := ctrl.Login(ctx, &TestLoginReq{Username: "admin", Password: "123456"})
	if resp.Token != "test-token-123" {
		t.Errorf("Unexpected token: %s", resp.Token)
	}
}

func TestModule_Dependency(t *testing.T) {
	globalContainer = NewContainer()
	globalModuleManager = &ModuleManager{
		modules: make(map[string]*Module),
		order:   make([]string, 0),
	}

	ctx := context.Background()

	// B depends on A
	moduleB := NewModule("sso").DependsOn("user-manager").OnInit(func(ctx context.Context) error { return nil })

	// Should fail without A
	err := UseModule(ctx, moduleB)
	if err == nil {
		t.Error("Should fail when dependency not registered")
	}

	// Register A first
	moduleA := NewModule("user-manager").OnInit(func(ctx context.Context) error {
		Provide[TestUserController]("dep.user.ctrl", &localTestUserController{})
		return nil
	})
	_ = UseModule(ctx, moduleA)

	// Now B should work
	moduleB2 := NewModule("sso").DependsOn("user-manager").OnInit(func(ctx context.Context) error { return nil })
	err = UseModule(ctx, moduleB2)
	if err != nil {
		t.Fatalf("Should succeed after dependency registered: %v", err)
	}
}
