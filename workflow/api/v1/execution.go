package v1

import "github.com/LandcLi/landc-go/frame/pkg/meta"

// ==================== 启动 ====================

type StartWorkflowRequest struct {
	meta.Meta   `path:"/api/workflows/:id/start" method:"POST"`
	WorkflowID  string `uri:"id" binding:"required"`
	TriggerType string `json:"trigger_type,omitempty"`
	TriggerID   string `json:"trigger_id,omitempty"`
	Input       string `json:"input,omitempty"`
}

type StartWorkflowResponse struct {
	ExecutionID string `json:"execution_id"`
}

// ==================== 查询执行 ====================

type GetExecutionRequest struct {
	meta.Meta `path:"/api/executions/:id" method:"GET"`
	ID        string `uri:"id" binding:"required"`
}

type GetExecutionResponse struct {
	ID           string  `json:"id"`
	WorkflowID   string  `json:"workflow_id"`
	WorkflowName string  `json:"workflow_name"`
	Status       string  `json:"status"`
	TriggerType  string  `json:"trigger_type"`
	Error        string  `json:"error,omitempty"`
	StartedAt    *string `json:"started_at,omitempty"`
	FinishedAt   *string `json:"finished_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

// ==================== 执行列表 ====================

type ListExecutionsRequest struct {
	meta.Meta   `path:"/api/executions" method:"GET"`
	WorkflowID  string `form:"workflow_id,omitempty"`
	Status      string `form:"status,omitempty"`
	Page        int    `form:"page,default=1"`
	Size        int    `form:"size,default=20"`
}

type ListExecutionsResponse struct {
	Total int64                  `json:"total"`
	Items []GetExecutionResponse `json:"items"`
}

// ==================== 暂停 ====================

type PauseExecutionRequest struct {
	meta.Meta `path:"/api/executions/:id/pause" method:"POST"`
	ID        string `uri:"id" binding:"required"`
}

type PauseExecutionResponse struct {
	Success bool `json:"success"`
}

// ==================== 恢复 ====================

type ResumeExecutionRequest struct {
	meta.Meta `path:"/api/executions/:id/resume" method:"POST"`
	ID        string `uri:"id" binding:"required"`
}

type ResumeExecutionResponse struct {
	Success bool `json:"success"`
}

// ==================== 取消 ====================

type CancelExecutionRequest struct {
	meta.Meta `path:"/api/executions/:id/cancel" method:"POST"`
	ID        string `uri:"id" binding:"required"`
}

type CancelExecutionResponse struct {
	Success bool `json:"success"`
}
