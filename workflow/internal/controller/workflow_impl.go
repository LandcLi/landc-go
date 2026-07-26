package controller

import (
	"context"
	"encoding/json"

	api "github.com/LandcLi/landc-go/workflow/api"
	v1 "github.com/LandcLi/landc-go/workflow/api/v1"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
	"github.com/LandcLi/landc-go/workflow/service"
)

type workflowController struct{}

func init() {
	api.WorkflowGateway.Provide(&workflowController{})
}

func (c *workflowController) Create(ctx context.Context, req *v1.CreateWorkflowRequest) (*v1.CreateWorkflowResponse, error) {
	wf, err := service.GetWorkflowService().Create(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateWorkflowResponse{WorkflowID: wf.ID}, nil
}

func (c *workflowController) Get(ctx context.Context, req *v1.GetWorkflowRequest) (*v1.GetWorkflowResponse, error) {
	wf, err := service.GetWorkflowService().Get(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return toWorkflowResponse(wf), nil
}

func (c *workflowController) List(ctx context.Context, req *v1.ListWorkflowsRequest) (*v1.ListWorkflowsResponse, error) {
	items, total, err := service.GetWorkflowService().List(ctx, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	resp := &v1.ListWorkflowsResponse{Total: total}
	for _, wf := range items {
		resp.Items = append(resp.Items, *toWorkflowResponse(wf))
	}
	return resp, nil
}

func (c *workflowController) Delete(ctx context.Context, req *v1.DeleteWorkflowRequest) (*v1.DeleteWorkflowResponse, error) {
	err := service.GetWorkflowService().Delete(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &v1.DeleteWorkflowResponse{Success: true}, nil
}

func (c *workflowController) Start(ctx context.Context, req *v1.StartWorkflowRequest) (*v1.StartWorkflowResponse, error) {
	var input json.RawMessage
	if req.Input != "" {
		input = json.RawMessage(req.Input)
	}
	triggerType := model.TriggerType(req.TriggerType)
	if triggerType == "" {
		triggerType = model.TriggerTypeApi
	}
	execID, err := service.GetWorkflowService().Start(ctx, req.WorkflowID, input, triggerType, req.TriggerID)
	if err != nil {
		return nil, err
	}
	return &v1.StartWorkflowResponse{ExecutionID: execID}, nil
}

func toWorkflowResponse(wf *model.Workflow) *v1.GetWorkflowResponse {
	r := &v1.GetWorkflowResponse{
		ID: wf.ID, Name: wf.Name, Description: wf.Description,
		Version: wf.Version, Status: string(wf.Status),
		CreatedAt: wf.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: wf.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	for _, n := range wf.Nodes {
		r.Nodes = append(r.Nodes, v1.Node{
			ID: n.ID, Name: n.Name, Type: string(n.Type),
			Timeout: n.Timeout, MaxRetries: n.MaxRetries,
			RetryMode: n.RetryMode, OrderNo: n.OrderNo,
		})
	}
	for _, e := range wf.Edges {
		r.Edges = append(r.Edges, v1.Edge{
			ID: e.ID, SourceID: e.SourceID, TargetID: e.TargetID,
			Label: e.Label, OrderNo: e.OrderNo,
		})
	}
	return r
}
