package di

import (
	"context"
	"testing"
)

// ============ Module Tests ============

type MockUserService interface {
	Login(username string) string
}

type localUserService struct{}

func (s *localUserService) Login(username string) string {
	return "local:" + username
}

type remoteUserService struct {
	baseURL string
}

func (s *remoteUserService) Login(username string) string {
	return "remote:" + s.baseURL + ":" + username
}

func TestProvideAndRequire(t *testing.T) {
	// Clean up
	globalContainer = NewContainer()

	// Register local implementation
	Provide[MockUserService]("user.service", &localUserService{})

	// Retrieve
	svc := Require[MockUserService]("user.service")
	result := svc.Login("alice")
	if result != "local:alice" {
		t.Errorf("Expected 'local:alice', got '%s'", result)
	}
}

func TestProvideIdempotent(t *testing.T) {
	// Clean up
	globalContainer = NewContainer()

	// First registration
	Provide[MockUserService]("user.service2", &localUserService{})

	// Second registration should be ignored (not overwrite)
	Provide[MockUserService]("user.service2", &remoteUserService{baseURL: "http://remote"})

	// Should still be local
	svc := Require[MockUserService]("user.service2")
	result := svc.Login("bob")
	if result != "local:bob" {
		t.Errorf("Expected 'local:bob', got '%s'", result)
	}
}

func TestOverride(t *testing.T) {
	// Clean up
	globalContainer = NewContainer()

	// Register local
	Provide[MockUserService]("user.service3", &localUserService{})

	// Override with remote
	Override[MockUserService]("user.service3", &remoteUserService{baseURL: "http://user-svc"})

	// Should be remote now
	svc := Require[MockUserService]("user.service3")
	result := svc.Login("charlie")
	if result != "remote:http://user-svc:charlie" {
		t.Errorf("Expected 'remote:http://user-svc:charlie', got '%s'", result)
	}
}

func TestResolve(t *testing.T) {
	// Clean up
	globalContainer = NewContainer()

	// Not registered
	_, err := Resolve[MockUserService]("nonexistent.service")
	if err == nil {
		t.Error("Should return error for nonexistent service")
	}

	// Register and resolve
	Provide[MockUserService]("user.service4", &localUserService{})
	svc, err := Resolve[MockUserService]("user.service4")
	if err != nil {
		t.Fatalf("Failed to resolve: %v", err)
	}
	if svc.Login("dave") != "local:dave" {
		t.Error("Unexpected result")
	}
}

func TestModule(t *testing.T) {
	// Clean up
	globalContainer = NewContainer()
	globalModuleManager = &ModuleManager{
		modules: make(map[string]*Module),
		order:   make([]string, 0),
	}

	ctx := context.Background()

	// Define a module (local mode)
	userModule := NewModule("user-manager").
		OnInit(func(ctx context.Context) error {
			Provide[MockUserService]("user.service5", &localUserService{})
			return nil
		}).
		OnClose(func() error {
			return nil
		})

	// Use the module
	err := UseModule(ctx, userModule)
	if err != nil {
		t.Fatalf("Failed to use module: %v", err)
	}

	// Verify service is available
	svc := Require[MockUserService]("user.service5")
	if svc.Login("eve") != "local:eve" {
		t.Error("Unexpected result")
	}
}

func TestModuleSwitchToRemote(t *testing.T) {
	// Clean up
	globalContainer = NewContainer()
	globalModuleManager = &ModuleManager{
		modules: make(map[string]*Module),
		order:   make([]string, 0),
	}

	ctx := context.Background()

	// First: local module
	localModule := NewModule("user-manager").
		OnInit(func(ctx context.Context) error {
			Provide[MockUserService]("user.service6", &localUserService{})
			return nil
		})

	err := UseModule(ctx, localModule)
	if err != nil {
		t.Fatalf("Failed to use local module: %v", err)
	}

	// Verify local
	svc := Require[MockUserService]("user.service6")
	if svc.Login("frank") != "local:frank" {
		t.Error("Expected local result")
	}

	// Switch to remote: Override the service
	Override[MockUserService]("user.service6", &remoteUserService{baseURL: "http://user-svc:8081"})

	// Verify remote
	svc = Require[MockUserService]("user.service6")
	result := svc.Login("frank")
	if result != "remote:http://user-svc:8081:frank" {
		t.Errorf("Expected remote result, got '%s'", result)
	}
}

func TestModuleDependency(t *testing.T) {
	// Clean up
	globalContainer = NewContainer()
	globalModuleManager = &ModuleManager{
		modules: make(map[string]*Module),
		order:   make([]string, 0),
	}

	ctx := context.Background()

	// Module B depends on Module A
	moduleB := NewModule("sso").
		DependsOn("user-manager").
		OnInit(func(ctx context.Context) error {
			return nil
		})

	// Try to use B without A -> should fail
	err := UseModule(ctx, moduleB)
	if err == nil {
		t.Error("Should fail when dependency is not registered")
	}

	// Register A first
	moduleA := NewModule("user-manager").
		OnInit(func(ctx context.Context) error {
			Provide[MockUserService]("user.service7", &localUserService{})
			return nil
		})

	err = UseModule(ctx, moduleA)
	if err != nil {
		t.Fatalf("Failed to use module A: %v", err)
	}

	// Now B should work
	moduleBRetry := NewModule("sso").
		DependsOn("user-manager").
		OnInit(func(ctx context.Context) error {
			return nil
		})

	err = UseModule(ctx, moduleBRetry)
	if err != nil {
		t.Fatalf("Failed to use module B after A: %v", err)
	}
}
