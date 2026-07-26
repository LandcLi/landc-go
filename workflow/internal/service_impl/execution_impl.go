package service_impl

import (
	"context"

	"github.com/LandcLi/landc-go/workflow/pkg/engine"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
	storepkg "github.com/LandcLi/landc-go/workflow/pkg/store"
	"github.com/LandcLi/landc-go/workflow/service"
)

type executionServiceImpl struct {
	store  storepkg.Store
	engine *engine.Engine
}

func init() {
	service.RegisterExecutionService(&executionServiceImpl{
		store:  storepkg.GetStore(),
		engine: engine.GetEngine(),
	})
}

func (s *executionServiceImpl) Get(ctx context.Context, execID string) (*model.Execution, error) {
	return s.store.GetExecution(ctx, execID)
}

func (s *executionServiceImpl) List(ctx context.Context, workflowID string, status string, page, size int) ([]*model.Execution, int64, error) {
	offset := (page - 1) * size
	return s.store.ListExecutions(ctx, workflowID, offset, size)
}

func (s *executionServiceImpl) Pause(ctx context.Context, execID string) error {
	return s.engine.PauseWorkflow(ctx, execID)
}

func (s *executionServiceImpl) Resume(ctx context.Context, execID string) error {
	return s.engine.ResumeWorkflow(ctx, execID)
}

func (s *executionServiceImpl) Cancel(ctx context.Context, execID string) error {
	return s.engine.CancelWorkflow(ctx, execID)
}

func (s *executionServiceImpl) GetTasks(ctx context.Context, execID string) ([]*model.Task, error) {
	return s.store.ListTasks(ctx, execID)
}
