package v1

import "github.com/LandcLi/landc-go/frame/pkg/meta"

type ListTasksRequest struct {
	meta.Meta   `path:"/api/executions/:id/tasks" method:"GET"`
	ExecutionID string `uri:"id" binding:"required"`
}

type TaskItem struct {
	ID         string `json:"id"`
	NodeName   string `json:"node_name"`
	NodeType   string `json:"node_type"`
	Status     string `json:"status"`
	RetryCount int    `json:"retry_count"`
	Error      string `json:"error,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
}

type ListTasksResponse struct {
	Items []TaskItem `json:"items"`
}
