package di

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LandcLi/landc-go/api/core"
	"github.com/LandcLi/landc-go/frame/pkg/trace"
)

// TestProxyCall_UnwrapData 验证 landc-go 统一包装响应（{"code":10000,...,"data":{...}}）
// 能被正确解包，业务数据落在 data 字段。
func TestProxyCall_UnwrapData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(core.Response{
			Code:    10000,
			Message: "success",
			Data:    TestLoginResp{Token: "t1", UserID: 1},
		})
	}))
	defer server.Close()

	client := NewProxyClient(server.URL)
	var resp TestLoginResp
	if err := client.dispatcher.call(context.Background(), "Login", &TestLoginReq{Username: "admin", Password: "123456"}, &resp); err != nil {
		t.Fatalf("call failed: %v", err)
	}
	if resp.Token != "t1" || resp.UserID != 1 {
		t.Errorf("unwrapped data mismatch: got %+v, want token=t1 user_id=1", resp)
	}
}

// TestProxyCall_RawResponse 验证非包装的裸对象响应保持兼容（直接反序列化整个 body）。
func TestProxyCall_RawResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"t2","user_id":2}`))
	}))
	defer server.Close()

	client := NewProxyClient(server.URL)
	var resp TestLoginResp
	if err := client.dispatcher.call(context.Background(), "Login", &TestLoginReq{Username: "admin", Password: "123456"}, &resp); err != nil {
		t.Fatalf("call failed: %v", err)
	}
	if resp.Token != "t2" || resp.UserID != 2 {
		t.Errorf("raw response mismatch: got %+v, want token=t2 user_id=2", resp)
	}
}

// TestProxyCall_DataNull 验证 data 为 null 的包装响应不误判（兼容路径，目标为零值）。
func TestProxyCall_DataNull(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":10000,"message":"success","data":null}`))
	}))
	defer server.Close()

	client := NewProxyClient(server.URL)
	var resp TestLoginResp
	if err := client.dispatcher.call(context.Background(), "Login", &TestLoginReq{Username: "admin", Password: "123456"}, &resp); err != nil {
		t.Fatalf("call failed: %v", err)
	}
	if resp.Token != "" || resp.UserID != 0 {
		t.Errorf("data:null should yield zero value, got %+v", resp)
	}
}

// TestProxyCall_PropagatesTraceID 跨服务调用必须带上上游的 trace_id。
//
// 缺陷背景：proxyDispatcher 只发静态配置的 header，从不读取 ctx。
// 一次 SSO → LUM 的调用，两边日志里的 trace_id 毫无关联，出问题时
// 无法把 A 的入口日志与 B 的异常日志串起来。
func TestProxyCall_PropagatesTraceID(t *testing.T) {
	const wantTrace = "trace-from-upstream"

	var gotTrace string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTrace = r.Header.Get("X-Trace-ID")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(core.Response{Code: 10000, Message: "success", Data: TestLoginResp{Token: "t", UserID: 1}})
	}))
	defer server.Close()

	client := NewProxyClient(server.URL)
	ctx := trace.WithTraceID(context.Background(), wantTrace)

	var resp TestLoginResp
	if err := client.dispatcher.call(ctx, "Login", &TestLoginReq{Username: "u", Password: "p"}, &resp); err != nil {
		t.Fatalf("call failed: %v", err)
	}
	if gotTrace != wantTrace {
		t.Errorf("下游收到的 X-Trace-ID = %q, want %q（trace 未透传）", gotTrace, wantTrace)
	}
}

// TestProxyCall_ExplicitHeaderOverridesTrace 显式配置的静态 header 优先于 ctx。
func TestProxyCall_ExplicitHeaderOverridesTrace(t *testing.T) {
	var gotTrace string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTrace = r.Header.Get("X-Trace-ID")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(core.Response{Code: 10000, Message: "success", Data: TestLoginResp{}})
	}))
	defer server.Close()

	client := NewProxyClient(server.URL, WithHeaders(map[string]string{"X-Trace-ID": "pinned"}))
	ctx := trace.WithTraceID(context.Background(), "from-ctx")

	var resp TestLoginResp
	if err := client.dispatcher.call(ctx, "Login", &TestLoginReq{}, &resp); err != nil {
		t.Fatalf("call failed: %v", err)
	}
	if gotTrace != "pinned" {
		t.Errorf("X-Trace-ID = %q, want pinned（显式配置应优先）", gotTrace)
	}
}
