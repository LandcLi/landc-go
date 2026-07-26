package store

import (
	"context"

	"github.com/LandcLi/landc-go/workflow/pkg/model"
)

// Store 工作流状态持久化接口
// 所有方法必须支持幂等语义
type Store interface {
	// ==================== 工作流定义 ====================

	CreateWorkflow(ctx context.Context, wf *model.Workflow) error
	UpdateWorkflow(ctx context.Context, wf *model.Workflow) error
	GetWorkflow(ctx context.Context, workflowID string) (*model.Workflow, error)
	GetWorkflowWithNodes(ctx context.Context, workflowID string) (*model.Workflow, error)
	ListWorkflows(ctx context.Context, offset, limit int) ([]*model.Workflow, int64, error)

	// ==================== 执行实例 ====================

	CreateExecution(ctx context.Context, exec *model.Execution) error
	// UpdateExecution 乐观锁更新, version 不匹配时返回 ErrVersionConflict
	UpdateExecution(ctx context.Context, exec *model.Execution) error
	GetExecution(ctx context.Context, execID string) (*model.Execution, error)
	ListExecutions(ctx context.Context, workflowID string, offset, limit int) ([]*model.Execution, int64, error)
	// GetExecutionByTriggerID 通过触发ID查找执行（用于幂等去重）
	GetExecutionByTriggerID(ctx context.Context, workflowID, triggerID string) (*model.Execution, error)

	// ==================== 任务实例 ====================

	CreateTask(ctx context.Context, task *model.Task) error
	UpdateTask(ctx context.Context, task *model.Task) error
	GetTask(ctx context.Context, taskID string) (*model.Task, error)
	ListTasks(ctx context.Context, execID string) ([]*model.Task, error)
	// ListTasksByStatus 按状态列出任务
	ListTasksByStatus(ctx context.Context, status model.TaskStatus, limit int) ([]*model.Task, error)
	// GetPendingTasks 获取待调度任务（含调度时间过滤）
	GetPendingTasks(ctx context.Context, nowTimestamp int64, limit int) ([]*model.Task, error)
	// CountPendingDependencies 统计指定执行中尚未完成的依赖任务数
	CountPendingDependencies(ctx context.Context, execID string, depNodeIDs []string) (int64, error)

	// ==================== Worker注册 ====================

	RegisterWorker(ctx context.Context, worker *model.Worker) error
	UpdateWorkerHeartbeat(ctx context.Context, workerID string) error
	ListActiveWorkers(ctx context.Context, group string) ([]*model.Worker, error)
	RemoveDeadWorkers(ctx context.Context, timeoutSeconds int64) error
}

// ErrVersionConflict 乐观锁版本冲突
type ErrVersionConflict struct {
	ID      string
	Current int
	Expect  int
}

func (e *ErrVersionConflict) Error() string {
	return "store: version conflict for " + e.ID
}

// ErrNotFound 记录不存在
type ErrNotFound struct {
	ID   string
	Type string
}

func (e *ErrNotFound) Error() string {
	return "store: " + e.Type + " not found: " + e.ID
}

// ErrDuplicateTriggerID 触发ID重复
type ErrDuplicateTriggerID struct {
	WorkflowID string
	TriggerID  string
}

func (e *ErrDuplicateTriggerID) Error() string {
	return "store: duplicate trigger_id " + e.TriggerID + " for workflow " + e.WorkflowID
}
