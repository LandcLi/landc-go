package store

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/LandcLi/landc-go/workflow/pkg/model"
)

// MemoryStore 基于内存的工作流状态存储（线程安全）
// 适用于单元测试、本地开发与轻量单机场景；生产环境请使用 DBStore
type MemoryStore struct {
	mu         sync.RWMutex
	workflows  map[string]*model.Workflow
	executions map[string]*model.Execution
	tasks      map[string]*model.Task
	workers    map[string]*model.Worker
}

// NewMemoryStore 创建内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		workflows:  make(map[string]*model.Workflow),
		executions: make(map[string]*model.Execution),
		tasks:      make(map[string]*model.Task),
		workers:    make(map[string]*model.Worker),
	}
}

// ==================== 工作流定义 ====================

func (s *MemoryStore) CreateWorkflow(ctx context.Context, wf *model.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.workflows[wf.ID]; exists {
		return &ErrDuplicateTriggerID{WorkflowID: wf.ID, TriggerID: ""}
	}
	s.workflows[wf.ID] = cloneWorkflow(wf)
	return nil
}

func (s *MemoryStore) UpdateWorkflow(ctx context.Context, wf *model.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.workflows[wf.ID]; !exists {
		return &ErrNotFound{ID: wf.ID, Type: "workflow"}
	}
	s.workflows[wf.ID] = cloneWorkflow(wf)
	return nil
}

func (s *MemoryStore) GetWorkflow(ctx context.Context, workflowID string) (*model.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wf, ok := s.workflows[workflowID]
	if !ok {
		return nil, &ErrNotFound{ID: workflowID, Type: "workflow"}
	}
	return cloneWorkflow(wf), nil
}

func (s *MemoryStore) GetWorkflowWithNodes(ctx context.Context, workflowID string) (*model.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wf, ok := s.workflows[workflowID]
	if !ok {
		return nil, &ErrNotFound{ID: workflowID, Type: "workflow"}
	}
	return cloneWorkflow(wf), nil
}

func (s *MemoryStore) ListWorkflows(ctx context.Context, offset, limit int) ([]*model.Workflow, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.workflows))
	for id := range s.workflows {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	total := int64(len(ids))
	if offset > len(ids) {
		offset = len(ids)
	}
	end := offset + limit
	if limit <= 0 || end > len(ids) {
		end = len(ids)
	}

	list := make([]*model.Workflow, 0, end-offset)
	for _, id := range ids[offset:end] {
		list = append(list, cloneWorkflow(s.workflows[id]))
	}
	return list, total, nil
}

// ==================== 执行实例 ====================

func (s *MemoryStore) CreateExecution(ctx context.Context, exec *model.Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.executions[exec.ID]; exists {
		return &ErrDuplicateTriggerID{WorkflowID: exec.WorkflowID, TriggerID: exec.TriggerID}
	}
	s.executions[exec.ID] = cloneExecution(exec)
	return nil
}

func (s *MemoryStore) UpdateExecution(ctx context.Context, exec *model.Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.executions[exec.ID]
	if !ok {
		return &ErrNotFound{ID: exec.ID, Type: "execution"}
	}
	// 乐观锁：version 不匹配时冲突
	if existing.Version != exec.Version {
		return &ErrVersionConflict{ID: exec.ID, Current: existing.Version, Expect: exec.Version}
	}
	exec.Version++
	s.executions[exec.ID] = cloneExecution(exec)
	return nil
}

func (s *MemoryStore) GetExecution(ctx context.Context, execID string) (*model.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exec, ok := s.executions[execID]
	if !ok {
		return nil, &ErrNotFound{ID: execID, Type: "execution"}
	}
	return cloneExecution(exec), nil
}

func (s *MemoryStore) ListExecutions(ctx context.Context, workflowID string, offset, limit int) ([]*model.Execution, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*model.Execution
	for _, exec := range s.executions {
		if workflowID == "" || exec.WorkflowID == workflowID {
			list = append(list, cloneExecution(exec))
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.After(list[j].CreatedAt) })
	total := int64(len(list))
	if offset > len(list) {
		offset = len(list)
	}
	end := offset + limit
	if limit <= 0 || end > len(list) {
		end = len(list)
	}
	return list[offset:end], total, nil
}

func (s *MemoryStore) GetExecutionByTriggerID(ctx context.Context, workflowID, triggerID string) (*model.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, exec := range s.executions {
		if exec.WorkflowID == workflowID && exec.TriggerID == triggerID {
			return cloneExecution(exec), nil
		}
	}
	return nil, nil
}

// ==================== 任务实例 ====================

func (s *MemoryStore) CreateTask(ctx context.Context, task *model.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = cloneTask(task)
	return nil
}

func (s *MemoryStore) UpdateTask(ctx context.Context, task *model.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[task.ID]; !ok {
		return &ErrNotFound{ID: task.ID, Type: "task"}
	}
	s.tasks[task.ID] = cloneTask(task)
	return nil
}

func (s *MemoryStore) GetTask(ctx context.Context, taskID string) (*model.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[taskID]
	if !ok {
		return nil, &ErrNotFound{ID: taskID, Type: "task"}
	}
	return cloneTask(task), nil
}

func (s *MemoryStore) ListTasks(ctx context.Context, execID string) ([]*model.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*model.Task
	for _, task := range s.tasks {
		if task.ExecutionID == execID {
			list = append(list, cloneTask(task))
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.Before(list[j].CreatedAt) })
	return list, nil
}

func (s *MemoryStore) ListTasksByStatus(ctx context.Context, status model.TaskStatus, limit int) ([]*model.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*model.Task
	for _, task := range s.tasks {
		if task.Status == status {
			list = append(list, cloneTask(task))
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.Before(list[j].CreatedAt) })
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (s *MemoryStore) GetPendingTasks(ctx context.Context, nowTimestamp int64, limit int) ([]*model.Task, error) {
	now := time.Unix(nowTimestamp, 0)
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*model.Task
	for _, task := range s.tasks {
		if task.Status == model.TaskStatusPending &&
			(task.ScheduledAt == nil || task.ScheduledAt.Before(now) || task.ScheduledAt.Equal(now)) {
			list = append(list, cloneTask(task))
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.Before(list[j].CreatedAt) })
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (s *MemoryStore) CountPendingDependencies(ctx context.Context, execID string, depNodeIDs []string) (int64, error) {
	if len(depNodeIDs) == 0 {
		return 0, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var count int64
	for _, task := range s.tasks {
		if task.ExecutionID != execID {
			continue
		}
		found := false
		for _, id := range depNodeIDs {
			if task.NodeID == id {
				found = true
				break
			}
		}
		if found && task.Status != model.TaskStatusCompleted && task.Status != model.TaskStatusSkipped {
			count++
		}
	}
	return count, nil
}

// ==================== Worker 注册 ====================

func (s *MemoryStore) RegisterWorker(ctx context.Context, worker *model.Worker) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workers[worker.ID] = cloneWorker(worker)
	return nil
}

func (s *MemoryStore) UpdateWorkerHeartbeat(ctx context.Context, workerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	worker, ok := s.workers[workerID]
	if !ok {
		return &ErrNotFound{ID: workerID, Type: "worker"}
	}
	worker.Heartbeat = time.Now()
	return nil
}

func (s *MemoryStore) ListActiveWorkers(ctx context.Context, group string) ([]*model.Worker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*model.Worker
	for _, w := range s.workers {
		if w.Status == "ACTIVE" && (group == "" || w.Group == group) {
			list = append(list, cloneWorker(w))
		}
	}
	return list, nil
}

func (s *MemoryStore) RemoveDeadWorkers(ctx context.Context, timeoutSeconds int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-time.Duration(timeoutSeconds) * time.Second)
	for id, w := range s.workers {
		if w.Heartbeat.Before(cutoff) {
			delete(s.workers, id)
		}
	}
	return nil
}

// ==================== 深拷贝辅助 ====================

func cloneWorkflow(wf *model.Workflow) *model.Workflow {
	if wf == nil {
		return nil
	}
	clone := *wf
	if wf.Nodes != nil {
		clone.Nodes = make([]*model.Node, len(wf.Nodes))
		for i, n := range wf.Nodes {
			if n == nil {
				continue
			}
			nc := *n
			clone.Nodes[i] = &nc
		}
	}
	if wf.Edges != nil {
		clone.Edges = make([]*model.Edge, len(wf.Edges))
		for i, e := range wf.Edges {
			if e == nil {
				continue
			}
			ec := *e
			clone.Edges[i] = &ec
		}
	}
	return &clone
}

func cloneExecution(exec *model.Execution) *model.Execution {
	if exec == nil {
		return nil
	}
	clone := *exec
	return &clone
}

func cloneTask(task *model.Task) *model.Task {
	if task == nil {
		return nil
	}
	clone := *task
	return &clone
}

func cloneWorker(w *model.Worker) *model.Worker {
	if w == nil {
		return nil
	}
	clone := *w
	return &clone
}
