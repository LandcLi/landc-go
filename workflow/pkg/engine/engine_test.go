package engine

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/workflow/pkg/executor"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
	"github.com/LandcLi/landc-go/workflow/pkg/observer"
	"github.com/LandcLi/landc-go/workflow/pkg/store"
)

// mockExecutor 测试执行器：包装任意函数
type mockExecutor struct {
	typeName string
	fn       func(ctx context.Context, req *executor.ExecuteRequest) (*executor.ExecuteResponse, error)
}

func (m *mockExecutor) Execute(ctx context.Context, req *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
	return m.fn(ctx, req)
}

func (m *mockExecutor) Type() string { return m.typeName }

func (m *mockExecutor) Schema() json.RawMessage { return nil }

// newTestEngine 构造带内存存储的 Engine
func newTestEngine(reg *executor.Registry) (*Engine, *store.MemoryStore) {
	ms := store.NewMemoryStore()
	obs := observer.NewObserverManager()
	eng := NewEngine(ms, reg, obs, nil, DefaultEngineConfig())
	return eng, ms
}

// newTestWorkflow 构造测试工作流（节点自动挂到 workflow）
func newTestWorkflow(id string, nodes []*model.Node, edges []*model.Edge) *model.Workflow {
	now := time.Now()
	wf := &model.Workflow{
		ID:        id,
		Name:      "test-" + id,
		Status:    model.WorkflowStatusActive,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	for _, n := range nodes {
		n.WorkflowID = id
		wf.Nodes = append(wf.Nodes, n)
	}
	for _, e := range edges {
		e.WorkflowID = id
		wf.Edges = append(wf.Edges, e)
	}
	return wf
}

// newCountExecutor 注册一个记录执行次数的 TEST 执行器
func newCountExecutor(reg *executor.Registry, counter *sync.Map) {
	reg.Register(&mockExecutor{
		typeName: "TEST",
		fn: func(ctx context.Context, req *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
			if v, ok := counter.Load(req.NodeID); ok {
				counter.Store(req.NodeID, v.(int)+1)
			} else {
				counter.Store(req.NodeID, 1)
			}
			return &executor.ExecuteResponse{Success: true, Output: json.RawMessage(`{"ok":true}`)}, nil
		},
	})
}

// waitForFinal 轮询执行状态直到终态
func waitForFinal(t *testing.T, ms *store.MemoryStore, execID string) *model.Execution {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		exec, err := ms.GetExecution(context.Background(), execID)
		if err != nil {
			t.Fatalf("get execution: %v", err)
		}
		if exec.IsFinal() {
			return exec
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("execution %s did not reach final state within timeout", execID)
	return nil
}

// TestEngine_SimpleDAG 验证 A→B→C 顺序链路全部成功执行
func TestEngine_SimpleDAG(t *testing.T) {
	reg := executor.NewRegistry()
	counter := &sync.Map{}
	newCountExecutor(reg, counter)

	eng, ms := newTestEngine(reg)

	wf := newTestWorkflow("wf-simple",
		[]*model.Node{
			{ID: "a", Name: "A", Type: "TEST", OrderNo: 1},
			{ID: "b", Name: "B", Type: "TEST", OrderNo: 2},
			{ID: "c", Name: "C", Type: "TEST", OrderNo: 3},
		},
		[]*model.Edge{
			{ID: "e1", SourceID: "a", TargetID: "b"},
			{ID: "e2", SourceID: "b", TargetID: "c"},
		},
	)
	if err := ms.CreateWorkflow(context.Background(), wf); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	execID, err := eng.StartWorkflow(context.Background(), "wf-simple", nil, model.TriggerTypeApi, "")
	if err != nil {
		t.Fatalf("start workflow: %v", err)
	}

	exec := waitForFinal(t, ms, execID)
	if exec.Status != model.ExecutionStatusCompleted {
		t.Fatalf("expected COMPLETED, got %s (err=%s)", exec.Status, exec.Error)
	}

	// 每个节点恰好执行一次
	for _, id := range []string{"a", "b", "c"} {
		if v, ok := counter.Load(id); !ok || v.(int) != 1 {
			t.Errorf("node %s should execute exactly once, got %v", id, v)
		}
	}
}

// TestEngine_ParallelNodes 验证并行分支（A→B,C→D）全部执行且无数据竞争
func TestEngine_ParallelNodes(t *testing.T) {
	reg := executor.NewRegistry()
	counter := &sync.Map{}
	newCountExecutor(reg, counter)

	eng, ms := newTestEngine(reg)

	wf := newTestWorkflow("wf-parallel",
		[]*model.Node{
			{ID: "a", Name: "A", Type: "TEST", OrderNo: 1},
			{ID: "b", Name: "B", Type: "TEST", OrderNo: 2},
			{ID: "c", Name: "C", Type: "TEST", OrderNo: 2},
			{ID: "d", Name: "D", Type: "TEST", OrderNo: 3},
		},
		[]*model.Edge{
			{ID: "e1", SourceID: "a", TargetID: "b"},
			{ID: "e2", SourceID: "a", TargetID: "c"},
			{ID: "e3", SourceID: "b", TargetID: "d"},
			{ID: "e4", SourceID: "c", TargetID: "d"},
		},
	)
	if err := ms.CreateWorkflow(context.Background(), wf); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	execID, err := eng.StartWorkflow(context.Background(), "wf-parallel", nil, model.TriggerTypeApi, "")
	if err != nil {
		t.Fatalf("start workflow: %v", err)
	}

	exec := waitForFinal(t, ms, execID)
	if exec.Status != model.ExecutionStatusCompleted {
		t.Fatalf("expected COMPLETED, got %s (err=%s)", exec.Status, exec.Error)
	}

	for _, id := range []string{"a", "b", "c", "d"} {
		if v, ok := counter.Load(id); !ok || v.(int) != 1 {
			t.Errorf("node %s should execute exactly once, got %v", id, v)
		}
	}
}

// TestEngine_FailurePropagation 验证中间节点失败导致执行 FAILED
func TestEngine_FailurePropagation(t *testing.T) {
	reg := executor.NewRegistry()
	counter := &sync.Map{}
	newCountExecutor(reg, counter)
	// 覆盖 b 节点为失败执行器
	reg.Register(&mockExecutor{
		typeName: "TEST_B",
		fn: func(ctx context.Context, req *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
			return nil, jsonErr("boom")
		},
	})

	eng, ms := newTestEngine(reg)

	wf := newTestWorkflow("wf-fail",
		[]*model.Node{
			{ID: "a", Name: "A", Type: "TEST", OrderNo: 1},
			{ID: "b", Name: "B", Type: "TEST_B", OrderNo: 2},
			{ID: "c", Name: "C", Type: "TEST", OrderNo: 3},
		},
		[]*model.Edge{
			{ID: "e1", SourceID: "a", TargetID: "b"},
			{ID: "e2", SourceID: "b", TargetID: "c"},
		},
	)
	if err := ms.CreateWorkflow(context.Background(), wf); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	execID, err := eng.StartWorkflow(context.Background(), "wf-fail", nil, model.TriggerTypeApi, "")
	if err != nil {
		t.Fatalf("start workflow: %v", err)
	}

	exec := waitForFinal(t, ms, execID)
	if exec.Status != model.ExecutionStatusFailed {
		t.Fatalf("expected FAILED, got %s", exec.Status)
	}
}

// TestEngine_SkipOnFailure 验证 skip_on_failure 节点失败时继续向下执行
func TestEngine_SkipOnFailure(t *testing.T) {
	reg := executor.NewRegistry()
	counter := &sync.Map{}
	newCountExecutor(reg, counter)
	reg.Register(&mockExecutor{
		typeName: "TEST_FAIL",
		fn: func(ctx context.Context, req *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
			return nil, jsonErr("boom")
		},
	})

	eng, ms := newTestEngine(reg)

	wf := newTestWorkflow("wf-skip",
		[]*model.Node{
			{ID: "a", Name: "A", Type: "TEST", OrderNo: 1},
			{ID: "b", Name: "B", Type: "TEST_FAIL", OrderNo: 2, SkipOnFailure: true},
			{ID: "c", Name: "C", Type: "TEST", OrderNo: 3},
		},
		[]*model.Edge{
			{ID: "e1", SourceID: "a", TargetID: "b"},
			{ID: "e2", SourceID: "b", TargetID: "c"},
		},
	)
	if err := ms.CreateWorkflow(context.Background(), wf); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	execID, err := eng.StartWorkflow(context.Background(), "wf-skip", nil, model.TriggerTypeApi, "")
	if err != nil {
		t.Fatalf("start workflow: %v", err)
	}

	exec := waitForFinal(t, ms, execID)
	if exec.Status != model.ExecutionStatusCompleted {
		t.Fatalf("expected COMPLETED (skip_on_failure), got %s (err=%s)", exec.Status, exec.Error)
	}

	if v, ok := counter.Load("c"); !ok || v.(int) != 1 {
		t.Errorf("node C should execute once after skip, got %v", v)
	}
}

// jsonErr 构造带错误的 json 包装
func jsonErr(msg string) error {
	return &jsonError{msg: msg}
}

type jsonError struct{ msg string }

func (e *jsonError) Error() string { return e.msg }

// TestEngine_Idempotency 验证相同 triggerID 只执行一次
func TestEngine_Idempotency(t *testing.T) {
	reg := executor.NewRegistry()
	counter := &sync.Map{}
	newCountExecutor(reg, counter)

	eng, ms := newTestEngine(reg)

	wf := newTestWorkflow("wf-idem",
		[]*model.Node{
			{ID: "a", Name: "A", Type: "TEST", OrderNo: 1},
		},
		nil,
	)
	if err := ms.CreateWorkflow(context.Background(), wf); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	ctx := context.Background()
	triggerID := "trigger-1"

	execID1, err := eng.StartWorkflow(ctx, "wf-idem", nil, model.TriggerTypeApi, triggerID)
	if err != nil {
		t.Fatalf("first start: %v", err)
	}
	exec1 := waitForFinal(t, ms, execID1)
	if exec1.Status != model.ExecutionStatusCompleted {
		t.Fatalf("expected COMPLETED, got %s", exec1.Status)
	}

	// 相同 triggerID 重复触发应返回已存在的执行
	execID2, err := eng.StartWorkflow(ctx, "wf-idem", nil, model.TriggerTypeApi, triggerID)
	if err != nil {
		t.Fatalf("second start: %v", err)
	}
	if execID2 != execID1 {
		t.Errorf("idempotency failed: expected same execution ID, got %s vs %s", execID2, execID1)
	}
}
