package di

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/meta"
)

// proxyFactories stores registered proxy factories keyed by gateway name.
// Generated code from 'landc gen proxy' registers factories via RegisterProxyFactory.
var proxyFactories sync.Map

// RegisterProxyFactory registers a proxy factory for a gateway.
// The generated proxy code calls this in init().
// After registration, ProvideRemote will use this factory to create
// the proxy and register it in the DI container.
func RegisterProxyFactory[T any](name string, factory func(*ProxyClient) T) {
	proxyFactories.Store(name, func(client *ProxyClient) interface{} {
		return factory(client)
	})
}

// RouteInfo defines the HTTP method and path for a controller method.
type RouteInfo struct {
	Method string // HTTP method: GET, POST, PUT, DELETE
	Path   string // URL path: /api/user/login
}

// Routes maps controller method names to their HTTP route info.
type Routes map[string]RouteInfo

// Gateway wraps a controller interface and provides:
//   - Automatic HTTP proxy generation via ProvideRemote + code generation
//   - DI integration with Provide/Override/Require for local mode
//
// Type parameter T must be an interface type.
type Gateway[T any] struct {
	name string
}

// NewGateway creates a new service gateway.
//
//	gw := di.NewGateway[UserController]("user.controller")
func NewGateway[T any](name string) *Gateway[T] {
	return &Gateway[T]{
		name: name,
	}
}

// Provide registers a local implementation into the DI container.
// Call this when integrating the service locally (in-process).
func (g *Gateway[T]) Provide(impl T) {
	Provide[T](g.name, impl)
}

// Override forcefully replaces the current implementation.
func (g *Gateway[T]) Override(impl T) {
	Override[T](g.name, impl)
}

// Get retrieves the registered implementation. Panics if not registered.
func (g *Gateway[T]) Get() T {
	return Require[T](g.name)
}

// ProvideRemote creates an HTTP proxy client and registers it via DI.
// Requires a proxy factory registered by generated code from 'landc gen proxy'.
// After this, Get() returns the remote proxy and callers use the exact same interface methods.
//
// To share a client across multiple gateways (same backend service):
//
//	client := di.NewProxyClient("http://user-service:8081")
//	user.UserGateway.ProvideRemote("http://user-service:8081", di.WithProxyClient(client))
//	common.CommonGateway.ProvideRemote("http://user-service:8081", di.WithProxyClient(client))
//
// Without WithProxyClient, each ProvideRemote creates a new client.
//
// Remote mode (in main.go):
//
//	user.UserGateway.ProvideRemote("http://user-service:8081")
//	ctrl := user.UserGateway.Get()   // same signature as local mode
//	resp, _ := ctrl.Login(ctx, req)  // same signature as local mode
//
// Local mode (in init()):
//
//	user.UserGateway.Provide(localImpl)
func (g *Gateway[T]) ProvideRemote(baseURL string, opts ...RemoteOption) {
	cfg := &remoteConfig{
		baseURL: baseURL,
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	var client *ProxyClient
	if cfg.existingProxy != nil {
		// Use explicitly provided client (shared across gateways)
		client = cfg.existingProxy
	} else {
		client = NewProxyClient(baseURL, opts...)
	}

	if factory, ok := proxyFactories.Load(g.name); ok {
		fn := factory.(func(*ProxyClient) interface{})
		proxy := fn(client)
		if impl, ok := proxy.(T); ok {
			Override[T](g.name, impl)
			return
		}
	}

	var ifaceName string
	var zero T
	if t := reflect.TypeOf(&zero).Elem(); t.Kind() == reflect.Interface {
		ifaceName = t.String()
	}
	panic(fmt.Sprintf(
		"Gateway[%s]: no proxy factory registered for interface %s.\n"+
			"Run: landc gen proxy -type %s -gateway-name %s",
		g.name, ifaceName, ifaceName, g.name,
	))
}

// parseRoutesFromInterface automatically discovers routes from interface T.
// It examines each method's second parameter (request struct) and extracts Meta tags.
//
// Example:
//
//	type UserController interface {
//	    Login(ctx context.Context, req *LoginReq) (*LoginResp, error)
//	}
//
//	type LoginReq struct {
//	    meta.Meta `path:"/api/user/login" method:"POST"`
//	    Username  string `json:"username"`
//	}
//
// The function will extract: routes["Login"] = {Method: "POST", Path: "/api/user/login"}
func parseRoutesFromInterface[T any]() Routes {
	var zero T
	ifaceType := reflect.TypeOf(&zero).Elem()

	// Ensure T is an interface
	if ifaceType.Kind() != reflect.Interface {
		panic(fmt.Sprintf("Gateway[T]: T must be an interface, got %v", ifaceType))
	}

	routes := make(Routes)

	// Iterate over interface methods
	for i := 0; i < ifaceType.NumMethod(); i++ {
		method := ifaceType.Method(i)
		methodType := method.Type

		// Method must have at least 1 parameter (context.Context)
		// and at most 2 parameters (context.Context, *Request)
		if methodType.NumIn() < 1 || methodType.NumIn() > 2 {
			continue
		}

		// If method has 2 parameters, the second one should be the request struct
		if methodType.NumIn() == 2 {
			reqType := methodType.In(1)

			// Handle pointer type
			if reqType.Kind() == reflect.Ptr {
				reqType = reqType.Elem()
			}

			// Must be a struct
			if reqType.Kind() != reflect.Struct {
				continue
			}

			// Look for Meta field
			metaField, hasMeta := reqType.FieldByName("Meta")
			if !hasMeta {
				continue
			}

			// Extract path and method from Meta tag
			path := metaField.Tag.Get("path")
			httpMethod := metaField.Tag.Get("method")

			if path != "" && httpMethod != "" {
				routes[method.Name] = RouteInfo{
					Method: httpMethod,
					Path:   path,
				}
			}
		}
	}

	return routes
}

// Redirector interface - if a response implements this, remote proxy will handle redirect.
type Redirector interface {
	StatusCode() int
	Location() string
}

// CookieSetter interface - if a response implements this, remote proxy will set cookies.
type CookieSetter interface {
	Cookies() []CookieInfo
}

// CookieInfo holds cookie parameters.
type CookieInfo struct {
	Name     string
	Value    string
	MaxAge   int
	Path     string
	Domain   string
	Secure   bool
	HttpOnly bool
}

// HeaderSetter interface - if a response implements this, remote proxy will set response headers.
type HeaderSetter interface {
	Headers() map[string]string
}

// ============ Remote Proxy ============

// RemoteOption configures the remote proxy client.
type RemoteOption func(*remoteConfig)

type remoteConfig struct {
	baseURL       string
	timeout       time.Duration
	httpClient    *http.Client
	headers       map[string]string
	existingProxy *ProxyClient // reuse existing client
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) RemoteOption {
	return func(c *remoteConfig) { c.timeout = d }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) RemoteOption {
	return func(c *remoteConfig) { c.httpClient = client }
}

// WithHeaders sets default headers for all requests.
func WithHeaders(headers map[string]string) RemoteOption {
	return func(c *remoteConfig) { c.headers = headers }
}

// WithProxyClient reuses an existing ProxyClient instead of creating a new one.
// Use this to share a single HTTP client across multiple gateways.
//
//	client := di.NewProxyClient("http://user-service:8081")
//	userGw.ProvideRemote("http://user-service:8081", di.WithProxyClient(client))
//	commonGw.ProvideRemote("http://user-service:8081", di.WithProxyClient(client))
func WithProxyClient(client *ProxyClient) RemoteOption {
	return func(c *remoteConfig) { c.existingProxy = client }
}

// proxyDispatcher handles HTTP calls for the proxy.
type proxyDispatcher struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
}

// call makes an HTTP request to the remote service.
// It dynamically resolves the route (path + method) from the request struct's meta.Meta tag.
func (p *proxyDispatcher) call(ctx context.Context, methodName string, req, respPtr interface{}) error {
	// Dynamically resolve route from request struct's meta.Meta tag
	reqValue := reflect.ValueOf(req)
	reqType := reqValue.Type()
	if reqType.Kind() == reflect.Ptr {
		reqType = reqType.Elem()
	}

	var httpMethod, path string
	if reqType.Kind() == reflect.Struct {
		metaData := meta.Data(reflect.New(reqType).Elem().Interface())
		if p, ok := metaData["path"].(string); ok {
			path = p
		}
		if m, ok := metaData["method"].(string); ok {
			httpMethod = m
		}
	}
	if path == "" {
		// Fallback: use method name as path
		path = "/" + strings.ToLower(methodName)
	}
	if httpMethod == "" {
		httpMethod = "POST"
	}

	url := p.baseURL + path

	// For GET requests, don't send body
	var httpReq *http.Request
	var err error
	if httpMethod == "GET" {
		httpReq, err = http.NewRequestWithContext(ctx, httpMethod, url, http.NoBody)
	} else {
		data, marshalErr := json.Marshal(req)
		if marshalErr != nil {
			return fmt.Errorf("marshal request failed: %w", marshalErr)
		}
		httpReq, err = http.NewRequestWithContext(ctx, httpMethod, url, bytes.NewBuffer(data))
	}
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range p.headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to extract error message
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("%s", errResp.Error)
		}
		return fmt.Errorf("remote service returned status %d: %s", resp.StatusCode, string(body))
	}

	if respPtr != nil {
		if err := json.Unmarshal(body, respPtr); err != nil {
			return fmt.Errorf("unmarshal response failed: %w", err)
		}
	}

	return nil
}

// ProxyClient is a generic HTTP client that can call remote services.
// Use this as a building block for SDK implementations.
//
// Example:
//
//	type remoteUserController struct {
//	    client *di.ProxyClient
//	}
//
//	func (r *remoteUserController) Login(ctx context.Context, req *LoginReq) (*LoginResp, error) {
//	    return di.Call[LoginResp](r.client, ctx, "Login", req)
//	}
type ProxyClient struct {
	dispatcher *proxyDispatcher
}

// NewProxyClient creates a ProxyClient for the given base URL.
// Routes are automatically discovered from the request struct's meta.Meta tags.
//
//	client := di.NewProxyClient("http://user-service:8081")
func NewProxyClient(baseURL string, opts ...RemoteOption) *ProxyClient {
	cfg := &remoteConfig{
		baseURL: baseURL,
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.timeout}
	}

	return &ProxyClient{
		dispatcher: &proxyDispatcher{
			baseURL:    cfg.baseURL,
			httpClient: httpClient,
			headers:    cfg.headers,
		},
	}
}

// Call makes a remote call to the specified method.
// This is the core building block for SDK implementations.
//
//	resp, err := di.Call[LoginResp](client, ctx, "Login", req)
func Call[Resp any](client *ProxyClient, ctx context.Context, methodName string, req interface{}) (*Resp, error) {
	resp := new(Resp)
	err := client.dispatcher.call(ctx, methodName, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CallVoid makes a remote call that doesn't return a response body.
func CallVoid(client *ProxyClient, ctx context.Context, methodName string, req interface{}) error {
	return client.dispatcher.call(ctx, methodName, req, nil)
}
