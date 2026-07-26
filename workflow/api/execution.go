package api

import (
	"context"

	v1 "github.com/LandcLi/landc-go/workflow/api/v1"
	"github.com/LandcLi/landc-go/frame/pkg/di"
)

// ExecutionController 执行管理接口
type ExecutionController interface {
	Get(ctx context.Context, req *v1.GetExecutionRequest) (*v1.GetExecutionResponse, error)
	List(ctx context.Context, req *v1.ListExecutionsRequest) (*v1.ListExecutionsResponse, error)
	Pause(ctx context.Context, req *v1.PauseExecutionRequest) (*v1.PauseExecutionResponse, error)
	Resume(ctx context.Context, req *v1.ResumeExecutionRequest) (*v1.ResumeExecutionResponse, error)
	Cancel(ctx context.Context, req *v1.CancelExecutionRequest) (*v1.CancelExecutionResponse, error)
	Tasks(ctx context.Context, req *v1.ListTasksRequest) (*v1.ListTasksResponse, error)
}

// ExecutionGateway 执行管理 Gateway
var ExecutionGateway = di.NewGateway[ExecutionController]("execution.controller")

// GetExecutionController 获取已注册的实现
func GetExecutionController() ExecutionController {
	return ExecutionGateway.Get()
}
