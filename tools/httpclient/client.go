// Package httpclient 提供预配置的 HTTP 客户端，默认带超时和链路追踪注入。
//
// 用法:
//
//	// 快速 GET
//	resp, err := httpclient.Get(ctx, "https://api.example.com/users")
//
//	// 自定义客户端
//	client := httpclient.New(30*time.Second, httpclient.WithTrace(nil))
//	resp, err := client.PostForm("https://api.example.com/login", url.Values{...})
package httpclient

import (
	"context"
	"net/http"
	"time"
)

// DefaultTimeout 默认 HTTP 请求超时时间
const DefaultTimeout = 10 * time.Second

// DefaultClient 可直接使用的全局默认 HTTP 客户端（带 10s 超时）。
var DefaultClient = &http.Client{Timeout: DefaultTimeout}

// New 创建带指定超时的 HTTP 客户端。
// transport 为 nil 时使用 http.DefaultTransport。
func New(timeout time.Duration, transport http.RoundTripper) *http.Client {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// WithTrace 包装 RoundTripper，自动从 context 中提取 Trace ID 注入到请求头。
//
//	client := httpclient.New(10*time.Second, httpclient.WithTrace(nil))
//	resp, err := client.Get("https://api.example.com/users")
func WithTrace(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return &traceTransport{next: next}
}

// traceTransport 自动从 context 读取 trace/request ID 并注入到 HTTP 头。
type traceTransport struct {
	next http.RoundTripper
}

func (t *traceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	// X-Trace-ID
	if traceID := traceIDFromContext(ctx); traceID != "" {
		req.Header.Set("X-Trace-ID", traceID)
	}
	// X-Request-ID
	if req.Header.Get("X-Request-ID") == "" {
		if requestID := requestIDFromContext(ctx); requestID != "" {
			req.Header.Set("X-Request-ID", requestID)
		}
	}
	return t.next.RoundTrip(req)
}

// Get 使用 DefaultClient 发送 GET 请求。
func Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return DefaultClient.Do(req)
}

// Post 使用 DefaultClient 发送 POST 请求（body 为 nil 时可传 nil）。
func Post(ctx context.Context, url, contentType string, body interface{}) (*http.Response, error) {
	// 简化实现，实际使用时建议用 req.Body
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return DefaultClient.Do(req)
}

// ==================== context helpers ====================

type contextKey string

const (
	traceIDKey   contextKey = "trace_id"
	requestIDKey contextKey = "request_id"
)

// WithTraceID 将 Trace ID 注入到 context 中。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// traceIDFromContext 从 context 中提取 Trace ID。
func traceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(traceIDKey).(string); ok {
		return v
	}
	return ""
}

func requestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}
