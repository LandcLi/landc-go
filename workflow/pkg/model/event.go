package model

import "time"

// ============================================================
// WorkflowEvent — 工作流执行事件（用于实时推送）
// ============================================================

// WorkflowEvent — 工作流执行事件（双通道推送 + 树形追踪支持）
type WorkflowEvent struct {
	Type      string `json:"type"`
	// node.started       / node.completed / node.error
	// node.skipped       / node.retrying / node.awaiting_input
	// workflow.started   / workflow.completed / workflow.failed
	// workflow.paused    / workflow.resumed   / workflow.cancelled
	ExecutionID string `json:"execution_id,omitempty"`
	NodeID      string `json:"node_id,omitempty"`
	NodeName    string `json:"node_name,omitempty"`
	NodeType    string `json:"node_type,omitempty"`
	Content     string `json:"content,omitempty"`
	Error       string `json:"error,omitempty"`

	// 树形追踪字段（REQ-002）
	ParentNodeID string `json:"parent_node_id,omitempty"` // 父节点ID（条件/循环节点）
	BranchID     string `json:"branch_id,omitempty"`      // 条件分支标识（如 "pass" / "revise"）
	IterationID  string `json:"iteration_id,omitempty"`   // 循环迭代 ID

	Timestamp  int64  `json:"timestamp"`
	Duration   int64  `json:"duration,omitempty"`
	Input      string `json:"input,omitempty"`
	Output     string `json:"output,omitempty"`
	RetryCount int    `json:"retry_count,omitempty"`
}

// NewWorkflowEvent 创建事件
func NewWorkflowEvent(eventType, executionID, nodeID, nodeName, nodeType, content string) *WorkflowEvent {
	return &WorkflowEvent{
		Type:        eventType,
		ExecutionID: executionID,
		NodeID:      nodeID,
		NodeName:    nodeName,
		NodeType:    nodeType,
		Content:     content,
		Timestamp:   time.Now().UnixMilli(),
	}
}

// NewWorkflowEventWithParent 创建带父节点信息的事件（树形追踪用）
func NewWorkflowEventWithParent(eventType, executionID, nodeID, nodeName, nodeType, content, parentNodeID, branchID, iterationID string) *WorkflowEvent {
	return &WorkflowEvent{
		Type:         eventType,
		ExecutionID:  executionID,
		NodeID:       nodeID,
		NodeName:     nodeName,
		NodeType:     nodeType,
		Content:      content,
		ParentNodeID: parentNodeID,
		BranchID:     branchID,
		IterationID:  iterationID,
		Timestamp:    time.Now().UnixMilli(),
	}
}

// TracingNode — 追踪树节点（供前端树形面板展示）
type TracingNode struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Status     string          `json:"status"`
	ElapsedMs  int64           `json:"elapsed_ms"`
	Input      interface{}     `json:"input,omitempty"`
	Output     interface{}     `json:"output,omitempty"`
	Error      string          `json:"error,omitempty"`
	Children   []*TracingNode  `json:"children,omitempty"`

	// 条件分支信息
	BranchID string `json:"branch_id,omitempty"`

	// 循环迭代
	Iterations []*TracingIteration `json:"iterations,omitempty"`
}

// TracingIteration — 循环迭代
type TracingIteration struct {
	Index int            `json:"index"`
	Nodes []*TracingNode `json:"nodes"`
}

// NodeTrace — 节点执行追踪记录
type NodeTrace struct {
	NodeID       string      `json:"node_id"`
	NodeType     string      `json:"node_type"`
	NodeName     string      `json:"node_name"`
	Input        interface{} `json:"input"`
	Output       interface{} `json:"output"`
	Duration     int64       `json:"duration_ms"`
	Error        string      `json:"error,omitempty"`
	ParentNodeID string      `json:"parent_node_id,omitempty"`
	BranchID     string      `json:"branch_id,omitempty"`
	IterationID  string      `json:"iteration_id,omitempty"`
	Timestamp    time.Time   `json:"timestamp"`
}
