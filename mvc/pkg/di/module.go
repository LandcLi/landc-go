// Package di provides dependency injection container with module support.
//
// Module Pattern:
// A Module represents a self-contained business unit (e.g., UserManager, SSO)
// that can be integrated locally (in-process) or accessed remotely (via SDK/HTTP).
//
// Usage:
//
//	// Define a service interface in your API layer
//	type UserService interface {
//	    Login(ctx context.Context, req *LoginReq) (*LoginResp, error)
//	}
//
//	// Register with a unique service name (typically in init() of the implementation package)
//	di.Provide[UserService]("user.service", NewLocalUserService())
//
//	// Retrieve anywhere (controller, other modules, etc.)
//	svc := di.Require[UserService]("user.service")
//	resp, err := svc.Login(ctx, req)
//
//	// To switch to remote mode, just replace the registration in main:
//	di.Provide[UserService]("user.service", sdk.NewRemoteUserService("http://user-svc:8081"))
//
// Module integration:
//
//	// Define a module
//	var UserModule = di.NewModule("user-manager").
//	    OnInit(func(ctx context.Context) error {
//	        // register local implementations
//	        di.Provide[UserService]("user.service", NewLocalUserService())
//	        di.Provide[UserDAO]("user.dao", NewUserDAO())
//	        return nil
//	    })
//
//	// In main.go - local mode:
//	app.UseModule(UserModule)
//
//	// In main.go - remote mode:
//	app.UseModule(di.NewModule("user-manager").OnInit(func(ctx context.Context) error {
//	    di.Provide[UserService]("user.service", sdk.NewRemoteUserService(cfg.UserServiceURL))
//	    return nil
//	}))
package di

import (
	"context"
	"fmt"
	"sync"
)

// Module represents a self-contained business unit that can be integrated
// locally (in-process direct call) or remotely (via SDK/HTTP).
// Switching between local and remote mode only requires changing the module
// registration in main, without modifying any business logic code.
type Module struct {
	name     string
	initFn   func(ctx context.Context) error
	closeFn  func() error
	deps     []string // dependent module names
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
// This is where you register service implementations (local or remote).
func (m *Module) OnInit(fn func(ctx context.Context) error) *Module {
	m.initFn = fn
	return m
}

// OnClose sets the cleanup function for the module.
func (m *Module) OnClose(fn func() error) *Module {
	m.closeFn = fn
	return m
}

// DependsOn declares module dependencies (other modules that must init first).
func (m *Module) DependsOn(moduleNames ...string) *Module {
	m.deps = append(m.deps, moduleNames...)
	return m
}

// Name returns the module name.
func (m *Module) Name() string {
	return m.name
}

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
func (m *Module) Inited() bool {
	return m.inited
}

// ============ Service Registry Shortcuts ============

// Provide registers a service implementation for the given interface type.
// If the service is already registered and overwrite is false, it does nothing (safe for init()).
// This is the primary way to register local or remote implementations.
//
// Example:
//
//	// Local mode (in init() of implementation package)
//	di.Provide[UserService]("user.service", NewLocalUserService())
//
//	// Remote mode (in main.go)
//	di.Provide[UserService]("user.service", sdk.NewRemoteUserService(url))
func Provide[T any](name string, impl T) {
	// Only register if not already registered (safe for init() pattern)
	if !Has(name) {
		_ = RegisterInterface[T](name, impl, false)
	}
}

// Override forcefully registers a service implementation, replacing any existing one.
// Use this in main.go to switch from local to remote mode.
//
// Example:
//
//	// Override local impl with remote SDK
//	di.Override[UserService]("user.service", sdk.NewRemoteUserService(url))
func Override[T any](name string, impl T) {
	_ = RegisterInterface[T](name, impl, true)
}

// Require retrieves a service implementation. Panics if not found.
// Use this in business logic code where the service must exist.
//
// Example:
//
//	svc := di.Require[UserService]("user.service")
//	resp, err := svc.Login(ctx, req)
func Require[T any](name string) T {
	return MustGetInterface[T](name)
}

// Resolve retrieves a service implementation. Returns error if not found.
// Use this when you want to handle the absence gracefully.
func Resolve[T any](name string) (T, error) {
	return GetInterface[T](name)
}

// ============ Module Manager ============

// ModuleManager manages module lifecycle (init order, dependency resolution).
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
func GetModuleManager() *ModuleManager {
	return globalModuleManager
}

// UseModule registers and initializes a module.
// If a module with the same name already exists, it will be replaced.
func UseModule(ctx context.Context, modules ...*Module) error {
	return globalModuleManager.Use(ctx, modules...)
}

// Use registers and initializes modules in order.
func (mm *ModuleManager) Use(ctx context.Context, modules ...*Module) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for _, m := range modules {
		// Check dependencies
		for _, dep := range m.deps {
			if _, exists := mm.modules[dep]; !exists {
				return fmt.Errorf("module %q depends on %q which is not registered", m.Name(), dep)
			}
		}

		// Initialize the module
		if err := m.Init(ctx); err != nil {
			return fmt.Errorf("failed to init module %q: %w", m.Name(), err)
		}

		// Register (replace if exists)
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

// GetModule returns a registered module by name.
func (mm *ModuleManager) GetModule(name string) (*Module, bool) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	m, exists := mm.modules[name]
	return m, exists
}
