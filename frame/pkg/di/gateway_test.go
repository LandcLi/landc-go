package di

import (
	"context"
	"fmt"
	"testing"

	"github.com/LandcLi/landc-go/frame/pkg/meta"
	"github.com/gin-gonic/gin"
)

// ============ Test Types ============

type TestLoginReq struct {
	meta.Meta `path:"/api/user/login" method:"POST"`
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

type TestLoginResp struct {
	Token  string `json:"token"`
	UserID uint   `json:"user_id"`
}

type TestGetUserReq struct {
	meta.Meta `path:"/api/user/get-info" method:"POST"`
	UserID    uint `json:"user_id" binding:"required"`
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

// ============ Tests ============

func TestGateway_LocalMode(t *testing.T) {
	// Clean up
	globalContainer = NewContainer()

	gin.SetMode(gin.TestMode)

	// Create gateway and provide local implementation
	gw := NewGateway[TestUserController]("test.user.controller")
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

func TestGateway_RemoteMode(t *testing.T) {
	// Clean up
	globalContainer = NewContainer()

	// Test: Create gateway and verify route auto-discovery
	gw := NewGateway[TestUserController]("test.user.controller3")

	// Verify routes are auto-discovered from interface
	routes := parseRoutesFromInterface[TestUserController]()

	if len(routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routes))
	}

	loginRoute, ok := routes["Login"]
	if !ok {
		t.Error("Expected Login route to be found")
	} else {
		if loginRoute.Method != "POST" {
			t.Errorf("Expected Login method POST, got %s", loginRoute.Method)
		}
		if loginRoute.Path != "/api/user/login" {
			t.Errorf("Expected Login path /api/user/login, got %s", loginRoute.Path)
		}
	}

	getUserRoute, ok := routes["GetUser"]
	if !ok {
		t.Error("Expected GetUser route to be found")
	} else {
		if getUserRoute.Method != "POST" {
			t.Errorf("Expected GetUser method POST, got %s", getUserRoute.Method)
		}
		if getUserRoute.Path != "/api/user/get-info" {
			t.Errorf("Expected GetUser path /api/user/get-info, got %s", getUserRoute.Path)
		}
	}

	// Verify gateway can provide local implementation
	gw.Provide(&localTestUserController{})
	ctrl := gw.Get()
	if ctrl == nil {
		t.Error("Expected to get controller implementation")
	}
}

func TestProxyClient_Integration(t *testing.T) {
	// This test demonstrates route auto-discovery
	// Clean up
	globalContainer = NewContainer()

	// === Server Side ===
	serverGw := NewGateway[TestUserController]("server.user")
	serverGw.Provide(&localTestUserController{})

	// Verify routes are auto-discovered
	routes := parseRoutesFromInterface[TestUserController]()

	if len(routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routes))
	}

	// Verify we can get the implementation
	ctrl := serverGw.Get()
	resp, err := ctrl.Login(context.Background(), &TestLoginReq{
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
