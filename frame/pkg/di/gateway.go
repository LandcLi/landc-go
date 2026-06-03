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

	"github.com/gin-gonic/gin"
)

// RouteInfo defines the HTTP method and path for a controller method.
type RouteInfo struct {
	Method       string        // HTTP method: GET, POST, PUT, DELETE
	Path         string        // URL path: /api/user/login
	CtxEnrichers []CtxEnricher // Optional context enrichers (extract headers, IP, etc.)
}

// Routes maps controller method names to their HTTP route info.
type Routes map[string]RouteInfo

// Gateway wraps a controller interface and provides:
//   - Automatic Gin handler generation (server side)
//   - Automatic HTTP proxy generation (client side)
//   - DI integration with Provide/Override/Require
//
// Type parameter T must be an interface type.
type Gateway[T any] struct {
	name   string
	routes Routes
}

// NewGateway creates a new service gateway.
//
//	gw := di.NewGateway[UserController]("user.controller", userRoutes)
func NewGateway[T any](name string, routes Routes) *Gateway[T] {
	return &Gateway[T]{
		name:   name,
		routes: routes,
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
//
//	gw.ProvideRemote("http://user-service:8081")
func (g *Gateway[T]) ProvideRemote(baseURL string, opts ...RemoteOption) {
	cfg := &remoteConfig{
		baseURL: baseURL,
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	proxy := newProxy[T](cfg, g.routes)
	Override[T](g.name, proxy)
}

// RegisterRoutes automatically generates Gin handlers for all routes
// and registers them to the given Gin engine/group.
// Each handler: bind JSON request -> call controller method -> return JSON response.
//
//	gw.RegisterRoutes(ginEngine)
func (g *Gateway[T]) RegisterRoutes(router gin.IRouter) {
	impl := Require[T](g.name)
	implValue := reflect.ValueOf(impl)

	for methodName, route := range g.routes {
		method := implValue.MethodByName(methodName)
		if !method.IsValid() {
			panic(fmt.Sprintf("gateway %s: method %s not found on implementation", g.name, methodName))
		}

		handler := createGatewayHandler(method, methodName, route)

		switch route.Method {
		case "GET":
			router.GET(route.Path, handler)
		case "POST":
			router.POST(route.Path, handler)
		case "PUT":
			router.PUT(route.Path, handler)
		case "DELETE":
			router.DELETE(route.Path, handler)
		case "PATCH":
			router.PATCH(route.Path, handler)
		default:
			router.POST(route.Path, handler)
		}
	}
}

// RegisterRoutesFiltered registers only the specified methods to the given router.
// This is useful for separating public and protected routes.
//
//	// Register only public routes
//	gw.RegisterRoutesFiltered(r, []string{"Login", "Register"})
//
//	// Register protected routes with auth middleware
//	protected := r.Group("").Use(authMiddleware)
//	gw.RegisterRoutesFiltered(protected, []string{"GetUserInfo", "UpdateUserInfo"})
func (g *Gateway[T]) RegisterRoutesFiltered(router gin.IRouter, methodNames []string) {
	impl := Require[T](g.name)
	implValue := reflect.ValueOf(impl)

	for _, methodName := range methodNames {
		route, ok := g.routes[methodName]
		if !ok {
			panic(fmt.Sprintf("gateway %s: no route defined for method %s", g.name, methodName))
		}

		method := implValue.MethodByName(methodName)
		if !method.IsValid() {
			panic(fmt.Sprintf("gateway %s: method %s not found on implementation", g.name, methodName))
		}

		handler := createGatewayHandler(method, methodName, route)

		switch route.Method {
		case "GET":
			router.GET(route.Path, handler)
		case "POST":
			router.POST(route.Path, handler)
		case "PUT":
			router.PUT(route.Path, handler)
		case "DELETE":
			router.DELETE(route.Path, handler)
		case "PATCH":
			router.PATCH(route.Path, handler)
		default:
			router.POST(route.Path, handler)
		}
	}
}

// createGatewayHandler creates a Gin handler that supports multiple method signatures:
//
// Standard:     func(ctx context.Context, req *XxxReq) (*XxxResp, error)
// Void:         func(ctx context.Context, req *XxxReq) error
// No-request:   func(ctx context.Context) (*XxxResp, error)
// Raw-return:   func(ctx context.Context, req *XxxReq) (string/int/..., error)
//
// The handler also supports context enrichment via RouteInfo.CtxEnrichers.
func createGatewayHandler(method reflect.Value, methodName string, route RouteInfo) gin.HandlerFunc {
	methodType := method.Type()
	numIn := methodType.NumIn()
	numOut := methodType.NumOut()

	// Validate: at least 1 input (ctx), at most 2 inputs (ctx, req)
	if numIn < 1 || numIn > 2 {
		panic(fmt.Sprintf("method %s must have 1 or 2 parameters (ctx or ctx+req)", methodName))
	}
	// Validate: at least 1 output (error), at most 2 outputs (resp, error)
	if numOut < 1 || numOut > 2 {
		panic(fmt.Sprintf("method %s must have 1 or 2 return values (error or resp+error)", methodName))
	}

	hasReqParam := numIn == 2
	hasRespReturn := numOut == 2

	var reqType reflect.Type
	if hasReqParam {
		reqType = methodType.In(1) // *XxxReq
		if reqType.Kind() == reflect.Ptr {
			reqType = reqType.Elem()
		}
	}

	return func(c *gin.Context) {
		// Build context with enrichers
		ctx := c.Request.Context()
		if len(route.CtxEnrichers) > 0 {
			for _, enricher := range route.CtxEnrichers {
				ctx = enricher(ctx, c)
			}
		}

		// Build call arguments
		args := []reflect.Value{reflect.ValueOf(ctx)}

		if hasReqParam {
			reqPtr := reflect.New(reqType)
			// Bind based on HTTP method
			var bindErr error
			if c.Request.Method == "GET" {
				bindErr = c.ShouldBindQuery(reqPtr.Interface())
			} else {
				bindErr = c.ShouldBindJSON(reqPtr.Interface())
			}
			if bindErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
				return
			}
			args = append(args, reqPtr)
		}

		// Call method
		results := method.Call(args)

		// Handle results based on signature
		if hasRespReturn {
			// (resp, error) pattern
			errVal := results[1]
			if !errVal.IsNil() {
				err := errVal.Interface().(error)
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			resp := results[0].Interface()

			// Check if response implements Redirector interface
			if redirector, ok := resp.(Redirector); ok {
				c.Redirect(redirector.StatusCode(), redirector.Location())
				return
			}

			// Check if response implements CookieSetter interface
			if setter, ok := resp.(CookieSetter); ok {
				for _, cookie := range setter.Cookies() {
					c.SetCookie(cookie.Name, cookie.Value, cookie.MaxAge, cookie.Path, cookie.Domain, cookie.Secure, cookie.HttpOnly)
				}
			}

			// Check if response implements HeaderSetter interface
			if setter, ok := resp.(HeaderSetter); ok {
				for k, v := range setter.Headers() {
					c.Header(k, v)
				}
			}

			c.JSON(http.StatusOK, resp)
		} else {
			// (error) pattern - void response
			errVal := results[0]
			if !errVal.IsNil() {
				err := errVal.Interface().(error)
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		}
	}
}

// CtxEnricher is a function that enriches the context from gin.Context.
// Use this to extract headers, IP, cookies, etc. and inject into context.
//
// Example:
//
//	func IPEnricher(ctx context.Context, c *gin.Context) context.Context {
//	    return context.WithValue(ctx, "IP", c.ClientIP())
//	}
type CtxEnricher func(ctx context.Context, c *gin.Context) context.Context

// Redirector interface - if a response implements this, Gateway will send a redirect.
type Redirector interface {
	StatusCode() int
	Location() string
}

// CookieSetter interface - if a response implements this, Gateway will set cookies.
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

// HeaderSetter interface - if a response implements this, Gateway will set response headers.
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

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request failed: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
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
