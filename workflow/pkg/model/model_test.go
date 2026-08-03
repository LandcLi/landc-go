package model

import (
	"encoding/json"
	"testing"
	"time"
)

// TestWorkflowIsValid 验证工作流定义合法性校验
func TestWorkflowIsValid(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wf := &Workflow{
			Name: "wf",
			Nodes: []*Node{
				{ID: "n1", Name: "N1", Type: NodeTypeScript},
				{ID: "n2", Name: "N2", Type: NodeTypeScript},
			},
		}
		if !wf.IsValid() {
			t.Fatal("expected valid workflow")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		wf := &Workflow{
			Nodes: []*Node{{ID: "n1"}},
		}
		if wf.IsValid() {
			t.Fatal("workflow with empty name should be invalid")
		}
	})

	t.Run("no nodes", func(t *testing.T) {
		wf := &Workflow{Name: "wf"}
		if wf.IsValid() {
			t.Fatal("workflow without nodes should be invalid")
		}
	})

	t.Run("empty node id", func(t *testing.T) {
		wf := &Workflow{
			Name:  "wf",
			Nodes: []*Node{{ID: ""}},
		}
		if wf.IsValid() {
			t.Fatal("workflow with empty node id should be invalid")
		}
	})

	t.Run("duplicate node id", func(t *testing.T) {
		wf := &Workflow{
			Name: "wf",
			Nodes: []*Node{
				{ID: "n1", Name: "N1"},
				{ID: "n1", Name: "N2"},
			},
		}
		if wf.IsValid() {
			t.Fatal("workflow with duplicate node ids should be invalid")
		}
	})
}

// TestExecutionIsFinal 验证执行终态判定
func TestExecutionIsFinal(t *testing.T) {
	final := []ExecutionStatus{
		ExecutionStatusCompleted,
		ExecutionStatusFailed,
		ExecutionStatusCancelled,
		ExecutionStatusTimeout,
	}
	for _, s := range final {
		if e := (&Execution{Status: s}); !e.IsFinal() {
			t.Errorf("status %s should be final", s)
		}
	}
	nonFinal := []ExecutionStatus{
		ExecutionStatusPending,
		ExecutionStatusRunning,
		ExecutionStatusPaused,
	}
	for _, s := range nonFinal {
		if e := (&Execution{Status: s}); e.IsFinal() {
			t.Errorf("status %s should not be final", s)
		}
	}
}

// TestExecutionCanTransition 验证执行状态机转移合法性
func TestExecutionCanTransition(t *testing.T) {
	cases := []struct {
		from, to ExecutionStatus
		want     bool
	}{
		// PENDING
		{ExecutionStatusPending, ExecutionStatusRunning, true},
		{ExecutionStatusPending, ExecutionStatusCancelled, true},
		{ExecutionStatusPending, ExecutionStatusCompleted, false},
		{ExecutionStatusPending, ExecutionStatusPaused, false},
		{ExecutionStatusPending, ExecutionStatusFailed, false},
		{ExecutionStatusPending, ExecutionStatusTimeout, false},
		// RUNNING
		{ExecutionStatusRunning, ExecutionStatusCompleted, true},
		{ExecutionStatusRunning, ExecutionStatusFailed, true},
		{ExecutionStatusRunning, ExecutionStatusPaused, true},
		{ExecutionStatusRunning, ExecutionStatusCancelled, true},
		{ExecutionStatusRunning, ExecutionStatusTimeout, true},
		{ExecutionStatusRunning, ExecutionStatusRunning, false},
		// PAUSED
		{ExecutionStatusPaused, ExecutionStatusRunning, true},
		{ExecutionStatusPaused, ExecutionStatusCancelled, true},
		{ExecutionStatusPaused, ExecutionStatusCompleted, false},
		{ExecutionStatusPaused, ExecutionStatusPaused, false},
		// 终态不可转移
		{ExecutionStatusCompleted, ExecutionStatusRunning, false},
		{ExecutionStatusCompleted, ExecutionStatusFailed, false},
		{ExecutionStatusFailed, ExecutionStatusRunning, false},
		{ExecutionStatusCancelled, ExecutionStatusRunning, false},
		{ExecutionStatusTimeout, ExecutionStatusRunning, false},
	}
	for _, c := range cases {
		e := &Execution{Status: c.from}
		if got := e.CanTransition(c.to); got != c.want {
			t.Errorf("CanTransition(%s -> %s) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

// TestTaskIsFinal 验证任务终态判定
func TestTaskIsFinal(t *testing.T) {
	final := []TaskStatus{
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusSkipped,
		TaskStatusCancelled,
	}
	for _, s := range final {
		if tk := (&Task{Status: s}); !tk.IsFinal() {
			t.Errorf("task status %s should be final", s)
		}
	}
	nonFinal := []TaskStatus{
		TaskStatusPending,
		TaskStatusRunning,
		TaskStatusPaused,
		TaskStatusRetrying,
	}
	for _, s := range nonFinal {
		if tk := (&Task{Status: s}); tk.IsFinal() {
			t.Errorf("task status %s should not be final", s)
		}
	}
}

// TestNodeJSONSerialization 验证节点 JSON 序列化往返
func TestNodeJSONSerialization(t *testing.T) {
	cfg := json.RawMessage(`{"url":"https://example.com"}`)
	in := json.RawMessage(`{"id":1}`)
	out := json.RawMessage(`{"result":"ok"}`)

	n := &Node{
		ID:            "n1",
		WorkflowID:    "wf-1",
		Name:          "N1",
		Type:          NodeTypeHttp,
		Config:        cfg,
		InputMapping:  in,
		OutputMapping: out,
		Timeout:       30,
		MaxRetries:    2,
		SkipOnFailure: true,
	}

	data, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal node: %v", err)
	}

	var got Node
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal node: %v", err)
	}
	if got.ID != n.ID || got.Type != n.Type || got.Name != n.Name {
		t.Errorf("roundtrip mismatch: %+v vs %+v", got, n)
	}
	if string(got.Config) != string(cfg) {
		t.Errorf("config mismatch: %s", got.Config)
	}
	if !got.SkipOnFailure {
		t.Error("skip_on_failure should survive roundtrip")
	}
}

// TestExecutionJSONSerialization 验证执行 JSON 序列化往返
func TestExecutionJSONSerialization(t *testing.T) {
	now := time.Now()
	e := &Execution{
		ID:            "exec-1",
		WorkflowID:    "wf-1",
		WorkflowName:  "WF",
		TriggerType:   TriggerTypeApi,
		Status:        ExecutionStatusRunning,
		CurrentNodeID: "n1",
		StartedAt:     &now,
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal execution: %v", err)
	}
	var got Execution
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal execution: %v", err)
	}
	if got.ID != e.ID || got.Status != e.Status || got.TriggerType != e.TriggerType {
		t.Errorf("roundtrip mismatch: %+v vs %+v", got, e)
	}
	if got.CurrentNodeID != "n1" {
		t.Errorf("current_node_id mismatch: %q", got.CurrentNodeID)
	}
}

// TestWorkflowEventSerialization 验证事件构造与 JSON 序列化
func TestWorkflowEventSerialization(t *testing.T) {
	ev := NewWorkflowEvent("node.started", "exec-1", "n1", "N1", string(NodeTypeScript), "run")
	if ev.Type != "node.started" || ev.ExecutionID != "exec-1" || ev.NodeID != "n1" {
		t.Fatalf("event fields mismatch: %+v", ev)
	}
	if ev.Timestamp == 0 {
		t.Fatal("timestamp should be set")
	}

	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	var got WorkflowEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if got.Type != ev.Type || got.NodeName != "N1" || got.Timestamp != ev.Timestamp {
		t.Errorf("event roundtrip mismatch: %+v vs %+v", got, ev)
	}
}

// TestWorkflowEventWithParent 验证树形追踪字段
func TestWorkflowEventWithParent(t *testing.T) {
	ev := NewWorkflowEventWithParent(
		"node.completed", "exec-1", "n2", "N2", string(NodeTypeCondition),
		"done", "n1", "pass", "iter-3",
	)
	if ev.ParentNodeID != "n1" || ev.BranchID != "pass" || ev.IterationID != "iter-3" {
		t.Errorf("tree fields mismatch: %+v", ev)
	}
}

// TestTableName 验证 GORM 表名映射
func TestTableName(t *testing.T) {
	if got := (&Workflow{}).TableName(); got != "wf_workflows" {
		t.Errorf("workflow table name: %q", got)
	}
	if got := (&Node{}).TableName(); got != "wf_nodes" {
		t.Errorf("node table name: %q", got)
	}
	if got := (&Edge{}).TableName(); got != "wf_edges" {
		t.Errorf("edge table name: %q", got)
	}
	if got := (&Execution{}).TableName(); got != "wf_executions" {
		t.Errorf("execution table name: %q", got)
	}
	if got := (&Task{}).TableName(); got != "wf_tasks" {
		t.Errorf("task table name: %q", got)
	}
	if got := (&Worker{}).TableName(); got != "wf_workers" {
		t.Errorf("worker table name: %q", got)
	}
}
