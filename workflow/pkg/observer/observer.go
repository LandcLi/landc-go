package observer

import (
	"context"

	"github.com/LandcLi/landc-go/log/facade"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
)

// ============================================================
// Observer 工作流执行观察者接口
// ============================================================

type Observer interface {
	OnExecutionStarted(ctx context.Context, exec *model.Execution)
	OnExecutionCompleted(ctx context.Context, exec *model.Execution, err error)
	OnExecutionPaused(ctx context.Context, exec *model.Execution)
	OnExecutionResumed(ctx context.Context, exec *model.Execution)
	OnExecutionCancelled(ctx context.Context, exec *model.Execution)
	OnTaskStarted(ctx context.Context, task *model.Task, node *model.Node)
	OnTaskCompleted(ctx context.Context, task *model.Task, node *model.Node, err error)
	OnTaskRetrying(ctx context.Context, task *model.Task, attempt int)
	OnTaskSkipped(ctx context.Context, task *model.Task, reason string)
}

// ============================================================
// ObserverManager 复合观察者
// ============================================================

type ObserverManager struct {
	observers []Observer
}

func NewObserverManager() *ObserverManager {
	return &ObserverManager{}
}

func (m *ObserverManager) Register(o Observer) {
	m.observers = append(m.observers, o)
}

func (m *ObserverManager) dispatch(fn func(Observer)) {
	for _, o := range m.observers {
		fn(o)
	}
}

func (m *ObserverManager) OnExecutionStarted(ctx context.Context, exec *model.Execution) {
	m.dispatch(func(o Observer) { o.OnExecutionStarted(ctx, exec) })
}

func (m *ObserverManager) OnExecutionCompleted(ctx context.Context, exec *model.Execution, err error) {
	m.dispatch(func(o Observer) { o.OnExecutionCompleted(ctx, exec, err) })
}

func (m *ObserverManager) OnExecutionPaused(ctx context.Context, exec *model.Execution) {
	m.dispatch(func(o Observer) { o.OnExecutionPaused(ctx, exec) })
}

func (m *ObserverManager) OnExecutionResumed(ctx context.Context, exec *model.Execution) {
	m.dispatch(func(o Observer) { o.OnExecutionResumed(ctx, exec) })
}

func (m *ObserverManager) OnExecutionCancelled(ctx context.Context, exec *model.Execution) {
	m.dispatch(func(o Observer) { o.OnExecutionCancelled(ctx, exec) })
}

func (m *ObserverManager) OnTaskStarted(ctx context.Context, task *model.Task, node *model.Node) {
	m.dispatch(func(o Observer) { o.OnTaskStarted(ctx, task, node) })
}

func (m *ObserverManager) OnTaskCompleted(ctx context.Context, task *model.Task, node *model.Node, err error) {
	m.dispatch(func(o Observer) { o.OnTaskCompleted(ctx, task, node, err) })
}

func (m *ObserverManager) OnTaskRetrying(ctx context.Context, task *model.Task, attempt int) {
	m.dispatch(func(o Observer) { o.OnTaskRetrying(ctx, task, attempt) })
}

func (m *ObserverManager) OnTaskSkipped(ctx context.Context, task *model.Task, reason string) {
	m.dispatch(func(o Observer) { o.OnTaskSkipped(ctx, task, reason) })
}

// ============================================================
// LogObserver — 复用 landc-go 的 log/facade
// ============================================================

type LogObserver struct {
	logger facade.Logger
}

// NewLogObserver 从 log/facade 的 Logger 创建观察者
func NewLogObserver(logger facade.Logger) *LogObserver {
	if logger == nil {
		logger = facade.GetLoggerWithName("workflow")
	}
	return &LogObserver{logger: logger}
}

func (o *LogObserver) log(ctx context.Context, level facade.LogLevel, msg string, fields ...facade.Field) {
	switch level {
	case facade.DebugLevel:
		o.logger.WithContext(ctx).Debug(msg, fields...)
	case facade.InfoLevel:
		o.logger.WithContext(ctx).Info(msg, fields...)
	case facade.WarnLevel:
		o.logger.WithContext(ctx).Warn(msg, fields...)
	case facade.ErrorLevel:
		o.logger.WithContext(ctx).Error(msg, fields...)
	}
}

func (o *LogObserver) OnExecutionStarted(ctx context.Context, exec *model.Execution) {
	o.log(ctx, facade.InfoLevel, "[wf] execution started",
		facade.Field{Key: "execution_id", Value: exec.ID},
		facade.Field{Key: "workflow_id", Value: exec.WorkflowID},
		facade.Field{Key: "workflow_name", Value: exec.WorkflowName},
		facade.Field{Key: "trigger_type", Value: string(exec.TriggerType)},
	)
}

func (o *LogObserver) OnExecutionCompleted(ctx context.Context, exec *model.Execution, err error) {
	level := facade.InfoLevel
	fields := []facade.Field{
		{Key: "execution_id", Value: exec.ID},
		{Key: "workflow_id", Value: exec.WorkflowID},
		{Key: "status", Value: string(exec.Status)},
	}
	if err != nil {
		level = facade.ErrorLevel
		fields = append(fields, facade.Field{Key: "error", Value: err.Error()})
	}
	o.log(ctx, level, "[wf] execution completed", fields...)
}

func (o *LogObserver) OnExecutionPaused(ctx context.Context, exec *model.Execution) {
	o.log(ctx, facade.InfoLevel, "[wf] execution paused",
		facade.Field{Key: "execution_id", Value: exec.ID},
		facade.Field{Key: "current_node", Value: exec.CurrentNodeID},
	)
}

func (o *LogObserver) OnExecutionResumed(ctx context.Context, exec *model.Execution) {
	o.log(ctx, facade.InfoLevel, "[wf] execution resumed",
		facade.Field{Key: "execution_id", Value: exec.ID},
	)
}

func (o *LogObserver) OnExecutionCancelled(ctx context.Context, exec *model.Execution) {
	o.log(ctx, facade.WarnLevel, "[wf] execution cancelled",
		facade.Field{Key: "execution_id", Value: exec.ID},
	)
}

func (o *LogObserver) OnTaskStarted(ctx context.Context, task *model.Task, node *model.Node) {
	o.log(ctx, facade.InfoLevel, "[wf] task started",
		facade.Field{Key: "task_id", Value: task.ID},
		facade.Field{Key: "node_name", Value: task.NodeName},
		facade.Field{Key: "node_type", Value: string(task.NodeType)},
		facade.Field{Key: "execution_id", Value: task.ExecutionID},
	)
}

func (o *LogObserver) OnTaskCompleted(ctx context.Context, task *model.Task, node *model.Node, err error) {
	level := facade.InfoLevel
	fields := []facade.Field{
		{Key: "task_id", Value: task.ID},
		{Key: "node_name", Value: task.NodeName},
		{Key: "retry_count", Value: task.RetryCount},
	}
	if err != nil {
		level = facade.ErrorLevel
		fields = append(fields, facade.Field{Key: "error", Value: err.Error()})
	}
	o.log(ctx, level, "[wf] task completed", fields...)
}

func (o *LogObserver) OnTaskRetrying(ctx context.Context, task *model.Task, attempt int) {
	o.log(ctx, facade.WarnLevel, "[wf] task retrying",
		facade.Field{Key: "task_id", Value: task.ID},
		facade.Field{Key: "node_name", Value: task.NodeName},
		facade.Field{Key: "attempt", Value: attempt},
		facade.Field{Key: "max_retries", Value: task.MaxRetries},
	)
}

func (o *LogObserver) OnTaskSkipped(ctx context.Context, task *model.Task, reason string) {
	o.log(ctx, facade.InfoLevel, "[wf] task skipped",
		facade.Field{Key: "task_id", Value: task.ID},
		facade.Field{Key: "node_name", Value: task.NodeName},
		facade.Field{Key: "reason", Value: reason},
	)
}

// ============================================================
// TraceObserver — 复用 frame/pkg/trace 做链路追踪
// ============================================================

type TraceObserver struct{}

func NewTraceObserver() *TraceObserver { return &TraceObserver{} }

func (o *TraceObserver) OnExecutionStarted(ctx context.Context, exec *model.Execution) {
	// trace 信息通过 ctx 传递，由 engine 注入 span
	_ = exec
}

func (o *TraceObserver) OnExecutionCompleted(_ context.Context, _ *model.Execution, _ error) {}
func (o *TraceObserver) OnExecutionPaused(_ context.Context, _ *model.Execution)             {}
func (o *TraceObserver) OnExecutionResumed(_ context.Context, _ *model.Execution)            {}
func (o *TraceObserver) OnExecutionCancelled(_ context.Context, _ *model.Execution)          {}
func (o *TraceObserver) OnTaskStarted(_ context.Context, _ *model.Task, _ *model.Node)       {}
func (o *TraceObserver) OnTaskCompleted(_ context.Context, _ *model.Task, _ *model.Node, _ error) {}
func (o *TraceObserver) OnTaskRetrying(_ context.Context, _ *model.Task, _ int)              {}
func (o *TraceObserver) OnTaskSkipped(_ context.Context, _ *model.Task, _ string)            {}
