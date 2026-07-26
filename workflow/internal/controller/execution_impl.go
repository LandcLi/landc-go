package controller

import (
	"context"

	api "github.com/LandcLi/landc-go/workflow/api"
	v1 "github.com/LandcLi/landc-go/workflow/api/v1"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
	"github.com/LandcLi/landc-go/workflow/service"
)

type executionController struct{}

func init() {
	api.ExecutionGateway.Provide(&executionController{})
}

func (c *executionController) Get(ctx context.Context, req *v1.GetExecutionRequest) (*v1.GetExecutionResponse, error) {
	exec, err := service.GetExecutionService().Get(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return toExecutionResponse(exec), nil
}

func (c *executionController) List(ctx context.Context, req *v1.ListExecutionsRequest) (*v1.ListExecutionsResponse, error) {
	items, total, err := service.GetExecutionService().List(ctx, req.WorkflowID, req.Status, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	resp := &v1.ListExecutionsResponse{Total: total}
	for _, exec := range items {
		resp.Items = append(resp.Items, *toExecutionResponse(exec))
	}
	return resp, nil
}

func (c *executionController) Pause(ctx context.Context, req *v1.PauseExecutionRequest) (*v1.PauseExecutionResponse, error) {
	if err := service.GetExecutionService().Pause(ctx, req.ID); err != nil {
		return nil, err
	}
	return &v1.PauseExecutionResponse{Success: true}, nil
}

func (c *executionController) Resume(ctx context.Context, req *v1.ResumeExecutionRequest) (*v1.ResumeExecutionResponse, error) {
	if err := service.GetExecutionService().Resume(ctx, req.ID); err != nil {
		return nil, err
	}
	return &v1.ResumeExecutionResponse{Success: true}, nil
}

func (c *executionController) Cancel(ctx context.Context, req *v1.CancelExecutionRequest) (*v1.CancelExecutionResponse, error) {
	if err := service.GetExecutionService().Cancel(ctx, req.ID); err != nil {
		return nil, err
	}
	return &v1.CancelExecutionResponse{Success: true}, nil
}

func (c *executionController) Tasks(ctx context.Context, req *v1.ListTasksRequest) (*v1.ListTasksResponse, error) {
	tasks, err := service.GetExecutionService().GetTasks(ctx, req.ExecutionID)
	if err != nil {
		return nil, err
	}
	resp := &v1.ListTasksResponse{}
	for _, t := range tasks {
		item := v1.TaskItem{
			ID: t.ID, NodeName: t.NodeName, NodeType: string(t.NodeType),
			Status: string(t.Status), RetryCount: t.RetryCount,
			Error: t.Error,
		}
		if t.StartedAt != nil {
			item.StartedAt = t.StartedAt.Format("2006-01-02T15:04:05Z")
		}
		if t.FinishedAt != nil {
			item.FinishedAt = t.FinishedAt.Format("2006-01-02T15:04:05Z")
		}
		resp.Items = append(resp.Items, item)
	}
	return resp, nil
}

func toExecutionResponse(exec *model.Execution) *v1.GetExecutionResponse {
	r := &v1.GetExecutionResponse{
		ID: exec.ID, WorkflowID: exec.WorkflowID,
		WorkflowName: exec.WorkflowName, Status: string(exec.Status),
		TriggerType: string(exec.TriggerType), Error: exec.Error,
		CreatedAt: exec.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if exec.StartedAt != nil {
		s := exec.StartedAt.Format("2006-01-02T15:04:05Z")
		r.StartedAt = &s
	}
	if exec.FinishedAt != nil {
		s := exec.FinishedAt.Format("2006-01-02T15:04:05Z")
		r.FinishedAt = &s
	}
	return r
}
