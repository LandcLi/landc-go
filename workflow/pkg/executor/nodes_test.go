package executor

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestConditionNodeFieldMode 验证条件节点 field+operator 模式
func TestConditionNodeFieldMode(t *testing.T) {
	e := NewConditionNodeExecutor()
	req := &ExecuteRequest{
		Config: json.RawMessage(`{"field":"status","operator":"equals","expected":"active"}`),
		Input:  json.RawMessage(`{"score":85,"status":"active"}`),
	}
	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success: %s", resp.Error)
	}
	if !strings.Contains(string(resp.Output), "true") {
		t.Errorf("expected true output, got %s", resp.Output)
	}
}

// TestConditionNodeFieldModeFalse 验证不满足条件返回 false
func TestConditionNodeFieldModeFalse(t *testing.T) {
	e := NewConditionNodeExecutor()
	req := &ExecuteRequest{
		Config: json.RawMessage(`{"field":"status","operator":"equals","expected":"active"}`),
		Input:  json.RawMessage(`{"score":30,"status":"disabled"}`),
	}
	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(string(resp.Output), "true") {
		t.Errorf("expected false output, got %s", resp.Output)
	}
}

// TestConditionNodeExpressionMode 验证 JS 表达式模式
func TestConditionNodeExpressionMode(t *testing.T) {
	e := NewConditionNodeExecutor()
	req := &ExecuteRequest{
		Config: json.RawMessage(`{"expression":"input.score > 60 && input.status == 'active'"}`),
		Input:  json.RawMessage(`{"score":85,"status":"active"}`),
	}
	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(string(resp.Output), "true") {
		t.Errorf("expected true output, got %s", resp.Output)
	}
}

// TestConditionNodeExpressionModeFalse 验证表达式结果为 false
func TestConditionNodeExpressionModeFalse(t *testing.T) {
	e := NewConditionNodeExecutor()
	req := &ExecuteRequest{
		Config: json.RawMessage(`{"expression":"input.score > 100"}`),
		Input:  json.RawMessage(`{"score":50}`),
	}
	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(string(resp.Output), "true") {
		t.Errorf("expected false output, got %s", resp.Output)
	}
}

// TestConditionNodeExpressionError 验证表达式错误时保守返回 false
func TestConditionNodeExpressionError(t *testing.T) {
	e := NewConditionNodeExecutor()
	req := &ExecuteRequest{
		Config: json.RawMessage(`{"expression":"this is not valid"}`),
		Input:  json.RawMessage(`{}`),
	}
	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(string(resp.Output), "true") {
		t.Errorf("invalid expression should not evaluate to true")
	}
}

// TestConditionNodeTruthMode 验证无配置时判断整体输入
func TestConditionNodeTruthMode(t *testing.T) {
	e := NewConditionNodeExecutor()

	// 空输入 → false
	req := &ExecuteRequest{Input: json.RawMessage(`""`)}
	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(string(resp.Output), "true") {
		t.Errorf("empty input should be false")
	}

	// 非空 → true
	req = &ExecuteRequest{Input: json.RawMessage(`{"x":1}`)}
	resp, err = e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(string(resp.Output), "true") {
		t.Errorf("non-empty input should be true")
	}
}

// TestInputOutputExecutors 验证 Input/Output 节点透传
func TestInputOutputExecutors(t *testing.T) {
	input := json.RawMessage(`{"hello":"world"}`)

	in := NewInputNodeExecutor()
	resp, err := in.Execute(context.Background(), &ExecuteRequest{Input: input})
	if err != nil || string(resp.Output) != string(input) {
		t.Errorf("input executor should pass through, got %s err=%v", resp.Output, err)
	}

	out := NewOutputNodeExecutor()
	resp, err = out.Execute(context.Background(), &ExecuteRequest{Input: input})
	if err != nil || string(resp.Output) != string(input) {
		t.Errorf("output executor should pass through, got %s err=%v", resp.Output, err)
	}

	// nil 输入 → 空对象
	resp, err = out.Execute(context.Background(), &ExecuteRequest{})
	if err != nil || string(resp.Output) != `{}` {
		t.Errorf("output executor nil input should return {}, got %s err=%v", resp.Output, err)
	}
}
