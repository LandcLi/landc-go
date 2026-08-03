package service

import (
	"context"
	"encoding/json"

	"github.com/LandcLi/landc-go/frame/pkg/di"
	v1 "github.com/LandcLi/landc-go/workflow/api/v1"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
)

// WorkflowService 工作流业务接口
type WorkflowService interface {
	Create(ctx context.Context, req *v1.CreateWorkflowRequest) (*model.Workflow, error)
	Get(ctx context.Context, id string) (*model.Workflow, error)
	List(ctx context.Context, page, size int) ([]*model.Workflow, int64, error)
	Delete(ctx context.Context, id string) error
	Start(ctx context.Context, workflowID string, input json.RawMessage, triggerType model.TriggerType, triggerID string) (string, error)
}

func RegisterWorkflowService(impl WorkflowService) {
	di.Provide[WorkflowService]("workflow.service", impl)
}

func GetWorkflowService() WorkflowService {
	return di.Require[WorkflowService]("workflow.service")
}
