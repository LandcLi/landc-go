package service_impl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LandcLi/landc-go/tools/generate"
	v1 "github.com/LandcLi/landc-go/workflow/api/v1"
	"github.com/LandcLi/landc-go/workflow/pkg/engine"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
	storepkg "github.com/LandcLi/landc-go/workflow/pkg/store"
	"github.com/LandcLi/landc-go/workflow/service"
)

type workflowServiceImpl struct {
	store  storepkg.Store
	engine *engine.Engine
}

func init() {
	service.RegisterWorkflowService(newWorkflowServiceImpl())
}

func newWorkflowServiceImpl() *workflowServiceImpl {
	return &workflowServiceImpl{
		store:  storepkg.GetStore(),
		engine: engine.GetEngine(),
	}
}

func (s *workflowServiceImpl) Create(ctx context.Context, req *v1.CreateWorkflowRequest) (*model.Workflow, error) {
	wfID := generate.UUID()
	wf := &model.Workflow{
		ID:          wfID,
		Name:        req.Name,
		Description: req.Description,
		Version:     1,
		Status:      model.WorkflowStatusActive,
		Timeout:     req.Timeout,
		MaxRetries:  req.MaxRetries,
	}
	for _, n := range req.Nodes {
		node := &model.Node{
			ID:         n.ID,
			WorkflowID: wfID,
			Name:       n.Name,
			Type:       model.NodeType(n.Type),
			Timeout:    n.Timeout,
			MaxRetries: n.MaxRetries,
			RetryMode:  n.RetryMode,
			OrderNo:    n.OrderNo,
		}
		if n.Config != "" {
			node.Config = json.RawMessage(n.Config)
		}
		wf.Nodes = append(wf.Nodes, node)
	}
	for _, e := range req.Edges {
		wf.Edges = append(wf.Edges, &model.Edge{
			ID: e.ID, WorkflowID: wfID,
			SourceID: e.SourceID, TargetID: e.TargetID,
			Label: e.Label, OrderNo: e.OrderNo,
		})
	}
	if !wf.IsValid() {
		return nil, fmt.Errorf("invalid workflow definition")
	}
	return wf, s.store.CreateWorkflow(ctx, wf)
}

func (s *workflowServiceImpl) Get(ctx context.Context, id string) (*model.Workflow, error) {
	return s.store.GetWorkflowWithNodes(ctx, id)
}

func (s *workflowServiceImpl) List(ctx context.Context, page, size int) ([]*model.Workflow, int64, error) {
	offset := (page - 1) * size
	return s.store.ListWorkflows(ctx, offset, size)
}

func (s *workflowServiceImpl) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (s *workflowServiceImpl) Start(ctx context.Context, workflowID string, input json.RawMessage, triggerType model.TriggerType, triggerID string) (string, error) {
	return s.engine.StartWorkflow(ctx, workflowID, input, triggerType, triggerID)
}
