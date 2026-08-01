package store

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/workflow/pkg/model"
)

func now() time.Time { return time.Now() }

func newTestWorkflow(id string) *model.Workflow {
	return &model.Workflow{
		ID:        id,
		Name:      "wf-" + id,
		Status:    model.WorkflowStatusActive,
		Version:   1,
		CreatedAt: now(),
		UpdatedAt: now(),
		Nodes: []*model.Node{
			{ID: "n1", Name: "N1", Type: model.NodeTypeScript, WorkflowID: id, OrderNo: 1},
		},
		Edges: []*model.Edge{},
	}
}

// TestMemoryStoreWorkflowCRUD 验证工作流定义 CRUD
func TestMemoryStoreWorkflowCRUD(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	wf := newTestWorkflow("wf-1")
	if err := s.CreateWorkflow(ctx, wf); err != nil {
		t.Fatalf("create: %v", err)
	}
	// 重复创建 → 错误
	if err := s.CreateWorkflow(ctx, wf); err == nil {
		t.Fatal("duplicate create should fail")
	}

	got, err := s.GetWorkflow(ctx, "wf-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "wf-wf-1" {
		t.Errorf("unexpected name: %s", got.Name)
	}

	got.Name = "renamed"
	if err := s.UpdateWorkflow(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}
	got2, _ := s.GetWorkflow(ctx, "wf-1")
	if got2.Name != "renamed" {
		t.Errorf("update not persisted: %s", got2.Name)
	}

	// 不存在的 ID
	if _, err := s.GetWorkflow(ctx, "missing"); err == nil {
		t.Fatal("get missing workflow should fail")
	}
}

// TestMemoryStoreExecution 验证执行实例创建/更新（乐观锁）
func TestMemoryStoreExecution(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	exec := &model.Execution{
		ID:         "exec-1",
		WorkflowID: "wf-1",
		Status:     model.ExecutionStatusPending,
		Version:    0,
		CreatedAt:  now(),
	}
	if err := s.CreateExecution(ctx, exec); err != nil {
		t.Fatalf("create execution: %v", err)
	}

	// 乐观锁：用错误的 version 更新 → 冲突
	stale := &model.Execution{ID: "exec-1", Status: model.ExecutionStatusRunning, Version: 99}
	if err := s.UpdateExecution(ctx, stale); err == nil {
		t.Fatal("stale version update should conflict")
	}

	// 用当前 version（创建后为 0）更新 → 成功且 version 递增
	cur := &model.Execution{ID: "exec-1", Status: model.ExecutionStatusRunning, Version: 0}
	if err := s.UpdateExecution(ctx, cur); err != nil {
		t.Fatalf("update execution: %v", err)
	}
	got, _ := s.GetExecution(ctx, "exec-1")
	if got.Version != 1 {
		t.Errorf("expected version 1, got %d", got.Version)
	}
}

// TestMemoryStoreTaskAndPending 验证任务管理与 pending 查询
func TestMemoryStoreTaskAndPending(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	task := &model.Task{
		ID:          "task-1",
		ExecutionID: "exec-1",
		NodeID:      "n1",
		Status:      model.TaskStatusPending,
		CreatedAt:   now(),
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// pending 查询（nowTimestamp 为未来时间，确保 scheduled 无影响）
	pending, err := s.GetPendingTasks(ctx, time.Now().Unix()+1000, 10)
	if err != nil {
		t.Fatalf("pending tasks: %v", err)
	}
	if len(pending) != 1 || pending[0].ID != "task-1" {
		t.Errorf("unexpected pending tasks: %v", pending)
	}

	// 完成任务后不再 pending
	task.Status = model.TaskStatusCompleted
	if err := s.UpdateTask(ctx, task); err != nil {
		t.Fatalf("update task: %v", err)
	}
	pending, _ = s.GetPendingTasks(ctx, time.Now().Unix()+1000, 10)
	if len(pending) != 0 {
		t.Errorf("completed task should not be pending: %v", pending)
	}
}

// TestMemoryStoreWorker 验证 worker 注册/心跳/清理
func TestMemoryStoreWorker(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	w := &model.Worker{
		ID:        "worker-1",
		Group:     "default",
		Status:    "ACTIVE",
		Heartbeat: now(),
	}
	if err := s.RegisterWorker(ctx, w); err != nil {
		t.Fatalf("register worker: %v", err)
	}
	if err := s.UpdateWorkerHeartbeat(ctx, "worker-1"); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}

	active, err := s.ListActiveWorkers(ctx, "default")
	if err != nil || len(active) != 1 {
		t.Fatalf("list active workers: %v err=%v", active, err)
	}
}

// TestMemoryStoreConcurrentSafety 验证并发读写无数据竞争
func TestMemoryStoreConcurrentSafety(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			wf := newTestWorkflow(string(rune('a' + n)))
			_ = s.CreateWorkflow(ctx, wf)
			_, _ = s.GetWorkflow(ctx, wf.ID)
			_, _, _ = s.ListWorkflows(ctx, 0, 100)
			_, _ = s.ListActiveWorkers(ctx, "")
		}(i)
	}
	wg.Wait()
}
