package observer

import (
	"context"
	"errors"
	"testing"

	"github.com/LandcLi/landc-go/log/facade"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
)

// ============================================================
// recordingObserver — 记录所有回调的测试观察者
// ============================================================

type recordingObserver struct {
	calls []string
}

func newRecordingObserver() *recordingObserver { return &recordingObserver{} }

func (r *recordingObserver) record(name string) { r.calls = append(r.calls, name) }

func (r *recordingObserver) OnExecutionStarted(_ context.Context, _ *model.Execution) {
	r.record("exec_started")
}
func (r *recordingObserver) OnExecutionCompleted(_ context.Context, _ *model.Execution, _ error) {
	r.record("exec_completed")
}
func (r *recordingObserver) OnExecutionPaused(_ context.Context, _ *model.Execution) {
	r.record("exec_paused")
}
func (r *recordingObserver) OnExecutionResumed(_ context.Context, _ *model.Execution) {
	r.record("exec_resumed")
}
func (r *recordingObserver) OnExecutionCancelled(_ context.Context, _ *model.Execution) {
	r.record("exec_canceled")
}
func (r *recordingObserver) OnTaskStarted(_ context.Context, _ *model.Task, _ *model.Node) {
	r.record("task_started")
}
func (r *recordingObserver) OnTaskCompleted(_ context.Context, _ *model.Task, _ *model.Node, _ error) {
	r.record("task_completed")
}
func (r *recordingObserver) OnTaskRetrying(_ context.Context, _ *model.Task, _ int) {
	r.record("task_retrying")
}
func (r *recordingObserver) OnTaskSkipped(_ context.Context, _ *model.Task, _ string) {
	r.record("task_skipped")
}

// ============================================================
// 测试
// ============================================================

func newTestExec() *model.Execution {
	return &model.Execution{ID: "exec-1", WorkflowID: "wf-1", WorkflowName: "WF", Status: model.ExecutionStatusPending}
}

func newTestTask() *model.Task {
	return &model.Task{ID: "task-1", ExecutionID: "exec-1", NodeName: "N1", NodeType: model.NodeTypeScript}
}

func newTestNode() *model.Node {
	return &model.Node{ID: "n1", Name: "N1", Type: model.NodeTypeScript}
}

// TestObserverManagerBroadcast 验证事件广播到所有注册观察者
func TestObserverManagerBroadcast(t *testing.T) {
	m := NewObserverManager()
	o1 := newRecordingObserver()
	o2 := newRecordingObserver()
	m.Register(o1)
	m.Register(o2)

	ctx := context.Background()
	exec := newTestExec()
	task := newTestTask()
	node := newTestNode()

	m.OnExecutionStarted(ctx, exec)
	m.OnExecutionCompleted(ctx, exec, nil)
	m.OnExecutionPaused(ctx, exec)
	m.OnExecutionResumed(ctx, exec)
	m.OnExecutionCancelled(ctx, exec)
	m.OnTaskStarted(ctx, task, node)
	m.OnTaskCompleted(ctx, task, node, errors.New("boom"))
	m.OnTaskRetrying(ctx, task, 1)
	m.OnTaskSkipped(ctx, task, "condition false")

	want := []string{
		"exec_started", "exec_completed", "exec_paused", "exec_resumed", "exec_canceled",
		"task_started", "task_completed", "task_retrying", "task_skipped",
	}
	for _, obs := range []*recordingObserver{o1, o2} {
		if len(obs.calls) != len(want) {
			t.Fatalf("observer got %d calls, want %d: %v", len(obs.calls), len(want), obs.calls)
		}
		for i, call := range want {
			if obs.calls[i] != call {
				t.Errorf("call[%d] = %q, want %q", i, obs.calls[i], call)
			}
		}
	}
}

// TestObserverManagerNoObservers 验证无观察者时不 panic
func TestObserverManagerNoObservers(t *testing.T) {
	m := NewObserverManager()
	ctx := context.Background()
	exec := newTestExec()
	task := newTestTask()
	node := newTestNode()

	m.OnExecutionStarted(ctx, exec)
	m.OnExecutionCompleted(ctx, exec, errors.New("x"))
	m.OnExecutionPaused(ctx, exec)
	m.OnExecutionResumed(ctx, exec)
	m.OnExecutionCancelled(ctx, exec)
	m.OnTaskStarted(ctx, task, node)
	m.OnTaskCompleted(ctx, task, node, nil)
	m.OnTaskRetrying(ctx, task, 2)
	m.OnTaskSkipped(ctx, task, "skip")
}

// TestLogObserver 验证 LogObserver 各事件回调不 panic（含 nil logger 回退）
func TestLogObserver(t *testing.T) {
	o := NewLogObserver(nil) // nil logger → 回退默认
	if o.logger == nil {
		t.Fatal("nil logger should fall back to default")
	}

	ctx := context.Background()
	exec := newTestExec()
	task := newTestTask()
	node := newTestNode()

	o.OnExecutionStarted(ctx, exec)
	o.OnExecutionCompleted(ctx, exec, nil)
	o.OnExecutionCompleted(ctx, exec, errors.New("boom"))
	o.OnExecutionPaused(ctx, exec)
	o.OnExecutionResumed(ctx, exec)
	o.OnExecutionCancelled(ctx, exec)
	o.OnTaskStarted(ctx, task, node)
	o.OnTaskCompleted(ctx, task, node, nil)
	o.OnTaskCompleted(ctx, task, node, errors.New("boom"))
	o.OnTaskRetrying(ctx, task, 3)
	o.OnTaskSkipped(ctx, task, "reason")
}

// TestLogObserverWithLogger 验证显式传入 logger
func TestLogObserverWithLogger(t *testing.T) {
	logger := facade.GetLoggerWithName("test.observer")
	o := NewLogObserver(logger)
	if o.logger != logger {
		t.Fatal("explicit logger should be kept")
	}
	o.OnExecutionStarted(context.Background(), newTestExec())
}

// TestTraceObserver 验证 TraceObserver 回调不 panic
func TestTraceObserver(t *testing.T) {
	o := NewTraceObserver()
	ctx := context.Background()
	exec := newTestExec()
	task := newTestTask()
	node := newTestNode()

	o.OnExecutionStarted(ctx, exec)
	o.OnExecutionCompleted(ctx, exec, nil)
	o.OnExecutionPaused(ctx, exec)
	o.OnExecutionResumed(ctx, exec)
	o.OnExecutionCancelled(ctx, exec)
	o.OnTaskStarted(ctx, task, node)
	o.OnTaskCompleted(ctx, task, node, nil)
	o.OnTaskRetrying(ctx, task, 1)
	o.OnTaskSkipped(ctx, task, "skip")
}
