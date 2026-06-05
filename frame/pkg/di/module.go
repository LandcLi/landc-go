// Package di provides dependency injection, module management, and service proxy
// for building microservices that support transparent local/remote switching.
//
// # Core Concept: Service Gateway
//
// A ServiceGateway wraps a Controller interface and provides:
//   - Automatic Gin HTTP handler generation (server side)
//   - Automatic HTTP proxy client generation (client side, for remote calls)
//   - DI registration with Provide/Override/Require pattern
//
// This eliminates the need to manually write:
//   - Gin handler functions that just bind JSON and call controller
//   - SDK client structs that just POST JSON and decode response
//   - Register/Get global variable boilerplate
//
// # Usage Pattern
//
// Step 1: Define interface and request/response structs in your API package:
//
//	type UserController interface {
//	    Login(ctx context.Context, req *LoginReq) (*LoginResp, error)
//	    Register(ctx context.Context, req *RegisterReq) (*RegisterResp, error)
//	}
//
// Step 2: Define route mapping:
//
//	var UserRoutes = di.Routes{
//	    "Login":    {Method: "POST", Path: "/api/user/login"},
//	    "Register": {Method: "POST", Path: "/api/user/register"},
//	}
//
// Step 3: Create a gateway and register to Gin (server side):
//
//	gw := di.NewGateway[UserController]("user.controller", UserRoutes)
//	gw.Provide(localImpl)           // register local implementation
//	gw.RegisterRoutes(ginEngine)    // auto-generate Gin handlers
//
// Step 4: In another service, use remote proxy (client side):
//
//	// Run 'landc gen proxy -type UserController -gateway-name user.controller' first
//	gw := di.NewGateway[UserController]("user.controller")
//	gw.ProvideRemote("http://user-service:8081")  // uses generated proxy code
//
// Step 5: Business code is the same regardless of local/remote:
//
//	ctrl := di.Require[UserController]("user.controller")
//	resp, err := ctrl.Login(ctx, req)
package di

import (
	"context"
	"fmt"
	"sync"
)

// ============ Module ============

// Module represents a self-contained business unit that can be integrated
// locally (in-process) or remotely (via HTTP proxy).
type Module struct {
	name     string
	initFn   func(ctx context.Context) error
	closeFn  func() error
	deps     []string
	initOnce sync.Once
	inited   bool
}

// NewModule creates a new module with the given name.
func NewModule(name string) *Module {
	return &Module{
		name: name,
		deps: make([]string, 0),
	}
}

// OnInit sets the initialization function for the module.
func (m *Module) OnInit(fn func(ctx context.Context) error) *Module {
	m.initFn = fn
	return m
}

// OnClose sets the cleanup function for the module.
func (m *Module) OnClose(fn func() error) *Module {
	m.closeFn = fn
	return m
}

// DependsOn declares module dependencies.
func (m *Module) DependsOn(moduleNames ...string) *Module {
	m.deps = append(m.deps, moduleNames...)
	return m
}

// Name returns the module name.
func (m *Module) Name() string { return m.name }

// Init initializes the module (only once, thread-safe).
func (m *Module) Init(ctx context.Context) error {
	var err error
	m.initOnce.Do(func() {
		if m.initFn != nil {
			err = m.initFn(ctx)
		}
		if err == nil {
			m.inited = true
		}
	})
	return err
}

// Close cleans up the module resources.
func (m *Module) Close() error {
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}

// Inited returns whether the module has been initialized.
func (m *Module) Inited() bool { return m.inited }

// ============ Service Registry Shortcuts ============

// Provide registers a service implementation (skip if already registered).
// Safe to call in init().
func Provide[T any](name string, impl T) {
	if !Has(name) {
		_ = RegisterInterface[T](name, impl, false)
	}
}

// Override forcefully registers a service implementation, replacing any existing one.
// Use in main() to switch from local to remote mode.
func Override[T any](name string, impl T) {
	_ = RegisterInterface[T](name, impl, true)
}

// Require retrieves a service implementation. Panics if not found.
func Require[T any](name string) T {
	return MustGetInterface[T](name)
}

// Resolve retrieves a service implementation. Returns error if not found.
func Resolve[T any](name string) (T, error) {
	return GetInterface[T](name)
}

// ============ Module Manager ============

// ModuleManager manages module lifecycle.
type ModuleManager struct {
	mu      sync.Mutex
	modules map[string]*Module
	order   []string
}

var globalModuleManager = &ModuleManager{
	modules: make(map[string]*Module),
	order:   make([]string, 0),
}

// GetModuleManager returns the global module manager.
func GetModuleManager() *ModuleManager { return globalModuleManager }

// UseModule registers and initializes modules.
func UseModule(ctx context.Context, modules ...*Module) error {
	return globalModuleManager.Use(ctx, modules...)
}

// Use registers and initializes modules in order.
func (mm *ModuleManager) Use(ctx context.Context, modules ...*Module) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for _, m := range modules {
		for _, dep := range m.deps {
			if _, exists := mm.modules[dep]; !exists {
				return fmt.Errorf("module %q depends on %q which is not registered", m.Name(), dep)
			}
		}

		if err := m.Init(ctx); err != nil {
			return fmt.Errorf("failed to init module %q: %w", m.Name(), err)
		}

		if _, exists := mm.modules[m.Name()]; !exists {
			mm.order = append(mm.order, m.Name())
		}
		mm.modules[m.Name()] = m
	}
	return nil
}

// Close closes all modules in reverse order.
func (mm *ModuleManager) Close() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for i := len(mm.order) - 1; i >= 0; i-- {
		name := mm.order[i]
		if m, exists := mm.modules[name]; exists {
			if err := m.Close(); err != nil {
				return fmt.Errorf("failed to close module %q: %w", name, err)
			}
		}
	}
	return nil
}
