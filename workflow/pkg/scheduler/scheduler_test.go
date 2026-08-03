package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/registry"
	"github.com/LandcLi/landc-go/workflow/pkg/engine"
	"github.com/LandcLi/landc-go/workflow/pkg/executor"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
	"github.com/LandcLi/landc-go/workflow/pkg/observer"
	"github.com/LandcLi/landc-go/workflow/pkg/store"
)

// ============================================================
// 测试辅助
// ============================================================

func newTestStore() *store.MemoryStore { return store.NewMemoryStore() }

func newTestScheduler(st store.Store) *Scheduler {
	eng := engine.NewEngine(
		st,
		executor.NewRegistry(),
		observer.NewObserverManager(),
		nil,
		engine.DefaultEngineConfig(),
	)
	return NewScheduler(
		st,
		eng,
		nil, // 走 DB 注册回退路径
		"worker-test-1",
		"default",
		"127.0.0.1:8080",
		DefaultSchedulerConfig(),
	)
}

func newPendingTask(execID string) *model.Task {
	return &model.Task{
		ID:          "task-" + execID,
		ExecutionID: execID,
		NodeID:      "n1",
		NodeName:    "N1",
		NodeType:    model.NodeTypeScript,
		Status:      model.TaskStatusPending,
	}
}

// ============================================================
// 生命周期
// ============================================================

// TestSchedulerLifecycle 验证 Start/Stop/IsRunning 生命周期
func TestSchedulerLifecycle(t *testing.T) {
	st := newTestStore()
	s := newTestScheduler(st)

	if s.IsRunning() {
		t.Fatal("scheduler should not be running before Start")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !s.IsRunning() {
		t.Fatal("scheduler should be running after Start")
	}

	// 重复 Start 报错
	if err := s.Start(ctx); err == nil {
		t.Fatal("second Start should fail")
	}

	s.Stop()
	if s.IsRunning() {
		t.Fatal("scheduler should not be running after Stop")
	}

	// 重复 Stop 安全
	s.Stop()
}

// TestSchedulerWorkerIDGeneration 验证默认 workerID 生成
func TestSchedulerWorkerIDGeneration(t *testing.T) {
	st := newTestStore()
	s := NewScheduler(
		st,
		nil,
		nil,
		"",
		"default",
		"",
		DefaultSchedulerConfig(),
	)
	if s.workerID == "" {
		t.Fatal("workerID should be auto-generated when empty")
	}
	if s.workerID == "wf-worker-" {
		t.Fatal("auto-generated workerID should include a unique suffix")
	}
}

// TestSchedulerDBWorkerRegistration 验证 DB 回退注册路径写入 worker
func TestSchedulerDBWorkerRegistration(t *testing.T) {
	st := newTestStore()
	s := newTestScheduler(st)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()

	workers, err := st.ListActiveWorkers(ctx, "default")
	if err != nil {
		t.Fatalf("ListActiveWorkers: %v", err)
	}
	found := false
	for _, w := range workers {
		if w.ID == s.workerID {
			found = true
			if w.Status != "ACTIVE" {
				t.Errorf("worker status = %q, want ACTIVE", w.Status)
			}
		}
	}
	if !found {
		t.Fatalf("worker %s not registered, got %+v", s.workerID, workers)
	}
}

// ============================================================
// 任务调度
// ============================================================

// TestSchedulerDispatchSkipsFinalExecution 验证终态执行的 pending 任务被跳过
func TestSchedulerDispatchSkipsFinalExecution(t *testing.T) {
	st := newTestStore()
	s := newTestScheduler(st)

	ctx := context.Background()

	finalExec := &model.Execution{
		ID:         "exec-final",
		WorkflowID: "wf-1",
		Status:     model.ExecutionStatusCompleted,
	}
	if err := st.CreateExecution(ctx, finalExec); err != nil {
		t.Fatalf("create execution: %v", err)
	}
	task := newPendingTask("exec-final")
	if err := st.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	s.pollAndDispatch()

	// 等待异步 dispatch 完成
	waitForTaskStatus(t, st, task.ID, model.TaskStatusSkipped)
}

// TestSchedulerDispatchSkipsPausedExecution 验证暂停执行的 pending 任务被跳过
func TestSchedulerDispatchSkipsPausedExecution(t *testing.T) {
	st := newTestStore()
	s := newTestScheduler(st)

	ctx := context.Background()

	pausedExec := &model.Execution{
		ID:         "exec-paused",
		WorkflowID: "wf-1",
		Status:     model.ExecutionStatusPaused,
	}
	if err := st.CreateExecution(ctx, pausedExec); err != nil {
		t.Fatalf("create execution: %v", err)
	}
	task := newPendingTask("exec-paused")
	if err := st.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	s.pollAndDispatch()

	waitForTaskStatus(t, st, task.ID, model.TaskStatusSkipped)
}

// TestSchedulerDispatchKeepsRunningExecution 验证运行中执行的任务保持 pending
func TestSchedulerDispatchKeepsRunningExecution(t *testing.T) {
	st := newTestStore()
	s := newTestScheduler(st)

	ctx := context.Background()

	runningExec := &model.Execution{
		ID:         "exec-running",
		WorkflowID: "wf-1",
		Status:     model.ExecutionStatusRunning,
	}
	if err := st.CreateExecution(ctx, runningExec); err != nil {
		t.Fatalf("create execution: %v", err)
	}
	task := newPendingTask("exec-running")
	if err := st.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	s.pollAndDispatch()
	time.Sleep(50 * time.Millisecond)

	got, err := st.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Status != model.TaskStatusPending {
		t.Errorf("task status = %s, want PENDING (running execution not skipped)", got.Status)
	}
}

// TestSchedulerDispatchMissingExecution 验证 execution 不存在时任务不受影响
func TestSchedulerDispatchMissingExecution(t *testing.T) {
	st := newTestStore()
	s := newTestScheduler(st)

	ctx := context.Background()
	task := newPendingTask("exec-missing")
	if err := st.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// 不应 panic
	s.pollAndDispatch()
	time.Sleep(30 * time.Millisecond)

	got, err := st.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Status != model.TaskStatusPending {
		t.Errorf("task status = %s, want PENDING", got.Status)
	}
}

// waitForTaskStatus 轮询等待任务达到目标状态
func waitForTaskStatus(t *testing.T, st *store.MemoryStore, taskID string, want model.TaskStatus) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		task, err := st.GetTask(ctx, taskID)
		if err != nil {
			t.Fatalf("GetTask: %v", err)
		}
		if task.Status == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("task %s status not reached %s in time", taskID, want)
}

// ============================================================
// etcd 注册路径（mock registry）
// ============================================================

type mockRegistry struct {
	mu          sync.Mutex
	regs        map[string]bool
	registerErr error
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{regs: make(map[string]bool)}
}

func (m *mockRegistry) Register(ctx context.Context, inst *registry.ServiceInstance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.registerErr != nil {
		return m.registerErr
	}
	m.regs[inst.ID] = true
	return nil
}

func (m *mockRegistry) Deregister(ctx context.Context, inst *registry.ServiceInstance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.regs, inst.ID)
	return nil
}

func (m *mockRegistry) GetService(ctx context.Context, name string) ([]*registry.ServiceInstance, error) {
	return nil, nil
}

func (m *mockRegistry) Watch(ctx context.Context, name string) (registry.Watcher, error) {
	return nil, nil
}

func (m *mockRegistry) Close() error { return nil }

func (m *mockRegistry) isRegistered(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.regs[id]
}

// TestSchedulerEtcdRegistration 验证 etcd 注册/注销路径
func TestSchedulerEtcdRegistration(t *testing.T) {
	st := newTestStore()
	reg := newMockRegistry()

	eng := engine.NewEngine(
		st,
		executor.NewRegistry(),
		observer.NewObserverManager(),
		nil,
		engine.DefaultEngineConfig(),
	)
	cfg := DefaultSchedulerConfig()
	s := NewScheduler(st, eng, reg, "worker-etcd-1", "default", "127.0.0.1:8080", cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !reg.isRegistered(s.workerID) {
		t.Fatal("worker should be registered via etcd")
	}

	s.Stop()
	if reg.isRegistered(s.workerID) {
		t.Fatal("worker should be deregistered after Stop")
	}
}

// TestSchedulerEtcdRegisterError 验证 etcd 注册失败时 Start 报错且不 running
func TestSchedulerEtcdRegisterError(t *testing.T) {
	st := newTestStore()
	reg := newMockRegistry()
	reg.registerErr = context.DeadlineExceeded

	cfg := DefaultSchedulerConfig()
	s := NewScheduler(st, nil, reg, "worker-etcd-2", "default", "", cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err == nil {
		t.Fatal("Start should fail when etcd registration fails")
	}
	if s.IsRunning() {
		t.Fatal("scheduler should not be running after failed Start")
	}
}
