package di

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LandcLi/landc-go/api/core"
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
