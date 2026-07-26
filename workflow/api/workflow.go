package api

import (
	"context"

	v1 "github.com/LandcLi/landc-go/workflow/api/v1"
	"github.com/LandcLi/landc-go/frame/pkg/di"
)

// WorkflowController 工作流管理接口
type WorkflowController interface {
	Create(ctx context.Context, req *v1.CreateWorkflowRequest) (*v1.CreateWorkflowResponse, error)
	Get(ctx context.Context, req *v1.GetWorkflowRequest) (*v1.GetWorkflowResponse, error)
	List(ctx context.Context, req *v1.ListWorkflowsRequest) (*v1.ListWorkflowsResponse, error)
	Delete(ctx context.Context, req *v1.DeleteWorkflowRequest) (*v1.DeleteWorkflowResponse, error)
	Start(ctx context.Context, req *v1.StartWorkflowRequest) (*v1.StartWorkflowResponse, error)
}

// WorkflowGateway 工作流管理 Gateway
var WorkflowGateway = di.NewGateway[WorkflowController]("workflow.controller")

// GetWorkflowController 获取已注册的实现
func GetWorkflowController() WorkflowController {
	return WorkflowGateway.Get()
}
