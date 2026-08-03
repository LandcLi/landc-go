package v1

import "github.com/LandcLi/landc-go/frame/pkg/meta"

// ==================== 创建 ====================

type CreateWorkflowRequest struct {
	meta.Meta   `path:"/api/workflows" method:"POST"`
	Name        string `json:"name" binding:"required,min=1,max=256"`
	Description string `json:"description,omitempty"`
	Nodes       []Node `json:"nodes" binding:"required,min=1"`
	Edges       []Edge `json:"edges"`
	Timeout     int64  `json:"timeout"`
	MaxRetries  int    `json:"max_retries"`
}

type Node struct {
	ID         string `json:"id" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Type       string `json:"type" binding:"required"`
	Config     string `json:"config,omitempty"`
	Timeout    int64  `json:"timeout,omitempty"`
	MaxRetries int    `json:"max_retries,omitempty"`
	RetryMode  string `json:"retry_mode,omitempty"`
	OrderNo    int    `json:"order_no"`
}

type Edge struct {
	ID       string `json:"id" binding:"required"`
	SourceID string `json:"source_id" binding:"required"`
	TargetID string `json:"target_id" binding:"required"`
	Label    string `json:"label,omitempty"`
	OrderNo  int    `json:"order_no"`
}

type CreateWorkflowResponse struct {
	WorkflowID string `json:"workflow_id"`
}

// ==================== 查询 ====================

type GetWorkflowRequest struct {
	meta.Meta `path:"/api/workflows/:id" method:"GET"`
	ID        string `uri:"id" binding:"required"`
}

type GetWorkflowResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     int    `json:"version"`
	Status      string `json:"status"`
	Nodes       []Node `json:"nodes"`
	Edges       []Edge `json:"edges"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ==================== 列表 ====================

type ListWorkflowsRequest struct {
	meta.Meta `path:"/api/workflows" method:"GET"`
	Page      int `form:"page,default=1"`
	Size      int `form:"size,default=20"`
}

type ListWorkflowsResponse struct {
	Total int64                 `json:"total"`
	Items []GetWorkflowResponse `json:"items"`
}

// ==================== 删除 ====================

type DeleteWorkflowRequest struct {
	meta.Meta `path:"/api/workflows/:id" method:"DELETE"`
	ID        string `uri:"id" binding:"required"`
}

type DeleteWorkflowResponse struct {
	Success bool `json:"success"`
}
