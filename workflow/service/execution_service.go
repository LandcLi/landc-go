package service

import (
	"context"

	"github.com/LandcLi/landc-go/frame/pkg/di"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
)

// ExecutionService 执行管理业务接口
type ExecutionService interface {
	Get(ctx context.Context, execID string) (*model.Execution, error)
	List(ctx context.Context, workflowID string, status string, page, size int) ([]*model.Execution, int64, error)
	Pause(ctx context.Context, execID string) error
	Resume(ctx context.Context, execID string) error
	Cancel(ctx context.Context, execID string) error
	GetTasks(ctx context.Context, execID string) ([]*model.Task, error)
}

func RegisterExecutionService(impl ExecutionService) {
	di.Provide[ExecutionService]("execution.service", impl)
}

func GetExecutionService() ExecutionService {
	return di.Require[ExecutionService]("execution.service")
}
