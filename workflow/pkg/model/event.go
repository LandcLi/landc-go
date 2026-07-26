package model

import "time"

// ============================================================
// WorkflowEvent — 工作流执行事件（用于实时推送）
// ============================================================

// WorkflowEvent 工作流执行事件。
// Observer 是同步回调，Event 是异步 channel 推送，两者互补。
type WorkflowEvent struct {
	Type      string `json:"type"`
	// node.started       — 节点开始执行
	// node.completed     — 节点执行成功
	// node.error         — 节点执行失败
	// node.skipped       — 节点被跳过（skip_on_failure）
	// node.retrying      — 节点重试
	// node.awaiting_input — 等待人类输入
	// workflow.completed — 工作流完成
	// workflow.failed    — 工作流失败
	// workflow.paused    — 工作流暂停
	// workflow.resumed   — 工作流恢复
	ExecutionID string `json:"execution_id,omitempty"`
	NodeID      string `json:"node_id,omitempty"`
	NodeName    string `json:"node_name,omitempty"`
	NodeType    string `json:"node_type,omitempty"`
	Content     string `json:"content,omitempty"`
	Error       string `json:"error,omitempty"`
	Timestamp   int64  `json:"timestamp"`
	Duration    int64  `json:"duration,omitempty"` // 毫秒
	Input       string `json:"input,omitempty"`
	Output      string `json:"output,omitempty"`
	RetryCount  int    `json:"retry_count,omitempty"`
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

// NodeTrace 节点执行追踪记录
type NodeTrace struct {
	NodeID    string      `json:"node_id"`
	NodeType  string      `json:"node_type"`
	NodeName  string      `json:"node_name"`
	Input     interface{} `json:"input"`
	Output    interface{} `json:"output"`
	Duration  int64       `json:"duration_ms"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}
