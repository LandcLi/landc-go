package executor

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestScriptExecutorBasic 验证 JS 脚本执行与 input 注入
func TestScriptExecutorBasic(t *testing.T) {
	e := NewScriptExecutor()
	req := &ExecuteRequest{
		Config: json.RawMessage(`{"lang":"js","script":"const v = input.amount * 2; v"}`),
		Input:  json.RawMessage(`{"amount":21}`),
	}

	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
	if strings.TrimSpace(string(resp.Output)) != "42" {
		t.Errorf("expected output 42, got %s", resp.Output)
	}
}

// TestScriptExecutorObjectReturn 验证返回对象被序列化为 JSON
func TestScriptExecutorObjectReturn(t *testing.T) {
	e := NewScriptExecutor()
	req := &ExecuteRequest{
		Config: json.RawMessage(`{"script":"({sum: input.a + input.b, ok: true})"}`),
		Input:  json.RawMessage(`{"a":1,"b":2}`),
	}

	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
	var out map[string]any
	if err := json.Unmarshal(resp.Output, &out); err != nil {
		t.Fatalf("output not valid json: %v (%s)", err, resp.Output)
	}
	if out["sum"] != float64(3) || out["ok"] != true {
		t.Errorf("unexpected output: %v", out)
	}
}

// TestScriptExecutorUnsupportedLang 验证不支持的语言返回明确错误
func TestScriptExecutorUnsupportedLang(t *testing.T) {
	e := NewScriptExecutor()
	req := &ExecuteRequest{
		Config: json.RawMessage(`{"lang":"python","script":"print(1)"}`),
	}

	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if resp.Success {
		t.Fatal("python should not be supported")
	}
	if !strings.Contains(resp.Error, "unsupported script language") {
		t.Errorf("unexpected error: %s", resp.Error)
	}
}

// TestScriptExecutorSyntaxError 验证脚本语法错误被捕获
func TestScriptExecutorSyntaxError(t *testing.T) {
	e := NewScriptExecutor()
	req := &ExecuteRequest{
		Config: json.RawMessage(`{"script":"this is not valid js"}`),
	}

	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if resp.Success {
		t.Fatal("syntax error should fail")
	}
}

// TestScriptExecutorTimeout 验证死循环脚本超时中断
func TestScriptExecutorTimeout(t *testing.T) {
	e := NewScriptExecutor()
	req := &ExecuteRequest{
		Config: json.RawMessage(`{"script":"while(true){}","timeout":1}`),
	}

	start := time.Now()
	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if resp.Success {
		t.Fatal("infinite loop should time out")
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("timeout not effective, took %v", elapsed)
	}
}

// TestSubWorkflowExecutorNotImplemented 验证子工作流执行器明确报错
func TestSubWorkflowExecutorNotImplemented(t *testing.T) {
	e := NewSubWorkflowExecutor()
	req := &ExecuteRequest{
		Config: json.RawMessage(`{"workflow_id":"wf-1"}`),
		Input:  json.RawMessage(`{"x":1}`),
	}

	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if resp.Success {
		t.Fatal("sub_workflow should report not implemented")
	}
	if !strings.Contains(resp.Error, "not implemented") {
		t.Errorf("unexpected error: %s", resp.Error)
	}
}

// TestIsBlockedAddress 验证 SSRF 地址识别
func TestIsBlockedAddress(t *testing.T) {
	blocked := []string{
		"127.0.0.1",
		"localhost",
		"10.0.0.5",
		"172.16.0.1",
		"192.168.1.1",
		"0.0.0.0",
		"100.64.0.1", // CGNAT
		"[::1]",
	}
	for _, host := range blocked {
		if !isBlockedAddress(host) {
			t.Errorf("expected %s to be blocked", host)
		}
	}

	// 公网地址不应被误伤（example.com 可解析）
	if isBlockedAddress("example.com") {
		t.Log("example.com resolved to blocked address (network dependent), skipping strict check")
	}
}
