package store

import (
	"context"
	"errors"
	"time"

	"github.com/LandcLi/landc-go/workflow/pkg/model"
	"gorm.io/gorm"
)

// DBStore GORM 实现 Store 接口
type DBStore struct {
	db *gorm.DB
}

// NewDBStore 创建数据库存储
func NewDBStore(db *gorm.DB) *DBStore {
	return &DBStore{db: db}
}

// AutoMigrate 自动建表
func (s *DBStore) AutoMigrate() error {
	return s.db.AutoMigrate(
		&model.Workflow{},
		&model.Node{},
		&model.Edge{},
		&model.Execution{},
		&model.Task{},
		&model.Worker{},
	)
}

// ==================== 工作流定义 ====================

func (s *DBStore) CreateWorkflow(ctx context.Context, wf *model.Workflow) error {
	return s.db.WithContext(ctx).Create(wf).Error
}

func (s *DBStore) UpdateWorkflow(ctx context.Context, wf *model.Workflow) error {
	return s.db.WithContext(ctx).Save(wf).Error
}

func (s *DBStore) GetWorkflow(ctx context.Context, workflowID string) (*model.Workflow, error) {
	var wf model.Workflow
	err := s.db.WithContext(ctx).First(&wf, "id = ?", workflowID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &ErrNotFound{ID: workflowID, Type: "workflow"}
	}
	return &wf, err
}

func (s *DBStore) GetWorkflowWithNodes(ctx context.Context, workflowID string) (*model.Workflow, error) {
	var wf model.Workflow
	err := s.db.WithContext(ctx).
		Preload("Nodes", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_no ASC")
		}).
		Preload("Edges", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_no ASC")
		}).
		First(&wf, "id = ?", workflowID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &ErrNotFound{ID: workflowID, Type: "workflow"}
	}
	return &wf, err
}

func (s *DBStore) ListWorkflows(ctx context.Context, offset, limit int) ([]*model.Workflow, int64, error) {
	var total int64
	if err := s.db.WithContext(ctx).Model(&model.Workflow{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.Workflow
	if err := s.db.WithContext(ctx).Offset(offset).Limit(limit).
		Preload("Nodes", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_no ASC")
		}).
		Preload("Edges").
		Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ==================== 执行实例 ====================

func (s *DBStore) CreateExecution(ctx context.Context, exec *model.Execution) error {
	return s.db.WithContext(ctx).Create(exec).Error
}

func (s *DBStore) UpdateExecution(ctx context.Context, exec *model.Execution) error {
	result := s.db.WithContext(ctx).
		Model(exec).
		Where("version = ?", exec.Version).
		Updates(map[string]interface{}{
			"status":          exec.Status,
			"output":          exec.Output,
			"state_data":      exec.StateData,
			"current_node_id": exec.CurrentNodeID,
			"error":           exec.Error,
			"started_at":      exec.StartedAt,
			"finished_at":     exec.FinishedAt,
			"expires_at":      exec.ExpiresAt,
			"version":         gorm.Expr("version + 1"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return &ErrVersionConflict{ID: exec.ID, Current: exec.Version, Expect: exec.Version}
	}
	exec.Version++
	return nil
}

func (s *DBStore) GetExecution(ctx context.Context, execID string) (*model.Execution, error) {
	var exec model.Execution
	err := s.db.WithContext(ctx).First(&exec, "id = ?", execID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &ErrNotFound{ID: execID, Type: "execution"}
	}
	return &exec, err
}

func (s *DBStore) ListExecutions(ctx context.Context, workflowID string, offset, limit int) ([]*model.Execution, int64, error) {
	var total int64
	query := s.db.WithContext(ctx).Model(&model.Execution{})
	if workflowID != "" {
		query = query.Where("workflow_id = ?", workflowID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.Execution
	q2 := s.db.WithContext(ctx).Order("created_at DESC").Offset(offset).Limit(limit)
	if workflowID != "" {
		q2 = q2.Where("workflow_id = ?", workflowID)
	}
	if err := q2.Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *DBStore) GetExecutionByTriggerID(ctx context.Context, workflowID, triggerID string) (*model.Execution, error) {
	var exec model.Execution
	err := s.db.WithContext(ctx).
		Where("workflow_id = ? AND trigger_id = ?", workflowID, triggerID).
		First(&exec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &exec, err
}

// ==================== 任务实例 ====================

func (s *DBStore) CreateTask(ctx context.Context, task *model.Task) error {
	return s.db.WithContext(ctx).Create(task).Error
}

func (s *DBStore) UpdateTask(ctx context.Context, task *model.Task) error {
	return s.db.WithContext(ctx).Save(task).Error
}

func (s *DBStore) GetTask(ctx context.Context, taskID string) (*model.Task, error) {
	var task model.Task
	err := s.db.WithContext(ctx).First(&task, "id = ?", taskID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &ErrNotFound{ID: taskID, Type: "task"}
	}
	return &task, err
}

func (s *DBStore) ListTasks(ctx context.Context, execID string) ([]*model.Task, error) {
	var list []*model.Task
	err := s.db.WithContext(ctx).
		Where("execution_id = ?", execID).
		Order("created_at ASC").
		Find(&list).Error
	return list, err
}

func (s *DBStore) ListTasksByStatus(ctx context.Context, status model.TaskStatus, limit int) ([]*model.Task, error) {
	var list []*model.Task
	err := s.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at ASC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (s *DBStore) GetPendingTasks(ctx context.Context, nowTimestamp int64, limit int) ([]*model.Task, error) {
	var list []*model.Task
	now := time.Unix(nowTimestamp, 0)
	err := s.db.WithContext(ctx).
		Where("status = ? AND (scheduled_at IS NULL OR scheduled_at <= ?)", model.TaskStatusPending, now).
		Order("created_at ASC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (s *DBStore) CountPendingDependencies(ctx context.Context, execID string, depNodeIDs []string) (int64, error) {
	if len(depNodeIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := s.db.WithContext(ctx).Model(&model.Task{}).
		Where("execution_id = ? AND node_id IN ? AND status NOT IN ?",
			execID, depNodeIDs,
			[]model.TaskStatus{model.TaskStatusCompleted, model.TaskStatusSkipped}).
		Count(&count).Error
	return count, err
}

// ==================== Worker 注册 ====================

func (s *DBStore) RegisterWorker(ctx context.Context, worker *model.Worker) error {
	return s.db.WithContext(ctx).Create(worker).Error
}

func (s *DBStore) UpdateWorkerHeartbeat(ctx context.Context, workerID string) error {
	return s.db.WithContext(ctx).Model(&model.Worker{}).
		Where("id = ?", workerID).
		Update("heartbeat", time.Now()).Error
}

func (s *DBStore) ListActiveWorkers(ctx context.Context, group string) ([]*model.Worker, error) {
	var list []*model.Worker
	query := s.db.WithContext(ctx).Where("status = 'ACTIVE'")
	if group != "" {
		query = query.Where("`group` = ?", group)
	}
	err := query.Find(&list).Error
	return list, err
}

func (s *DBStore) RemoveDeadWorkers(ctx context.Context, timeoutSeconds int64) error {
	return s.db.WithContext(ctx).
		Where("heartbeat < ?", time.Now().Add(-time.Duration(timeoutSeconds)*time.Second)).
		Delete(&model.Worker{}).Error
}
