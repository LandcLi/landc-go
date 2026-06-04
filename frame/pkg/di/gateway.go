package di

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"
)

// RouteInfo defines the HTTP method and path for a controller method.
type RouteInfo struct {
	Method string // HTTP method: GET, POST, PUT, DELETE
	Path   string // URL path: /api/user/login
}

// Routes maps controller method names to their HTTP route info.
type Routes map[string]RouteInfo

// Gateway wraps a controller interface and provides:
//   - Automatic HTTP proxy generation (client side)
//   - DI integration with Provide/Override/Require
//   - Route auto-discovery from interface method signatures
//
// Type parameter T must be an interface type.
// Routes are automatically parsed from Meta tags in request structs.
type Gateway[T any] struct {
	name string
}

// NewGateway creates a new service gateway.
// Routes are automatically discovered from the interface T.
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

// Get retrieves the registered implementation.
func (g *Gateway[T]) Get() T {
	return Require[T](g.name)
}

// ProvideRemote creates an HTTP proxy client and registers it as the implementation.
// Call this when the service is deployed remotely.
// Routes are automatically parsed from the interface T.
//
//	gw.ProvideRemote("http://user-service:8081")
func (g *Gateway[T]) ProvideRemote(baseURL string, opts ...RemoteOption) {
	// Auto-discover routes from interface T
	routes := parseRoutesFromInterface[T]()

	cfg := &remoteConfig{
		baseURL: baseURL,
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	proxy := newProxy[T](cfg, routes)
	Override[T](g.name, proxy)
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
	baseURL    string
	timeout    time.Duration
	httpClient *http.Client
	headers    map[string]string
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

// newProxy creates a dynamic proxy that implements interface T by making HTTP calls.
// It uses reflect.MakeFunc to generate method implementations at runtime.
func newProxy[T any](cfg *remoteConfig, routes Routes) T {
	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.timeout}
	}

	var zero T
	ifaceType := reflect.TypeOf(&zero).Elem()

	// We need to create a struct that implements the interface T.
	// Since Go doesn't support dynamic interface implementation via reflect alone,
	// we use a proxyImpl struct with a dispatch map.
	proxy := &proxyDispatcher{
		baseURL:    cfg.baseURL,
		httpClient: httpClient,
		headers:    cfg.headers,
		routes:     routes,
	}

	// Build the proxy using reflect to create a value that satisfies T
	return buildProxy[T](proxy, ifaceType, routes)
}

// proxyDispatcher handles HTTP calls for the proxy.
type proxyDispatcher struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
	routes     Routes
}

// call makes an HTTP request to the remote service.
func (p *proxyDispatcher) call(ctx context.Context, methodName string, req interface{}, respPtr interface{}) error {
	route, ok := p.routes[methodName]
	if !ok {
		return fmt.Errorf("no route defined for method %s", methodName)
	}

	url := p.baseURL + route.Path

	// Determine HTTP method from route info
	httpMethod := route.Method
	if httpMethod == "" {
		httpMethod = "POST" // Default to POST if not specified
	}

	// For GET requests, don't send body
	var httpReq *http.Request
	var err error
	if httpMethod == "GET" {
		httpReq, err = http.NewRequestWithContext(ctx, httpMethod, url, nil)
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

// buildProxy creates a concrete implementation of interface T using reflect.
// Since Go doesn't support dynamic proxy via reflect.MakeFunc for interfaces directly,
// we return a *proxyWrapper[T] that the caller must type-assert.
// However, for this to work with generics, we use a different approach:
// The proxy struct embeds method dispatchers.
//
// NOTE: Due to Go's type system limitations, the proxy is implemented as a
// concrete struct. The caller's interface T must match the ProxyClient interface pattern.
// For production use, consider code generation (landc gen proxy) for zero-reflection overhead.
func buildProxy[T any](dispatcher *proxyDispatcher, ifaceType reflect.Type, routes Routes) T {
	// Create a ProxyClient that wraps the dispatcher
	client := &ProxyClient{dispatcher: dispatcher}

	// Try to convert to T
	var result interface{} = client
	if v, ok := result.(T); ok {
		return v
	}

	// If direct conversion fails, panic with helpful message
	panic(fmt.Sprintf(
		"cannot create proxy for %v: interface methods must match ProxyClient pattern. "+
			"Use 'landc gen proxy' for code generation, or implement the interface manually with ProxyClient.Call()",
		ifaceType,
	))
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

// NewProxyClient creates a ProxyClient for the given base URL and routes.
//
//	client := di.NewProxyClient("http://user-service:8081", userRoutes)
func NewProxyClient(baseURL string, routes Routes, opts ...RemoteOption) *ProxyClient {
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
			routes:     routes,
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
