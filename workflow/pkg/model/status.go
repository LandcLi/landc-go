package model

// ============================================================
// 通用状态枚举
// ============================================================

// WorkflowStatus 工作流定义状态
type WorkflowStatus string

const (
	WorkflowStatusActive   WorkflowStatus = "ACTIVE"
	WorkflowStatusPaused   WorkflowStatus = "PAUSED"
	WorkflowStatusArchived WorkflowStatus = "ARCHIVED"
)

// ExecutionStatus 工作流执行实例状态
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "PENDING"
	ExecutionStatusRunning   ExecutionStatus = "RUNNING"
	ExecutionStatusPaused    ExecutionStatus = "PAUSED"
	ExecutionStatusCompleted ExecutionStatus = "COMPLETED"
	ExecutionStatusFailed    ExecutionStatus = "FAILED"
	ExecutionStatusCancelled ExecutionStatus = "CANCELLED"
	ExecutionStatusTimeout   ExecutionStatus = "TIMEOUT"
)

// TaskStatus 任务（节点执行实例）状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "PENDING"
	TaskStatusRunning   TaskStatus = "RUNNING"
	TaskStatusPaused    TaskStatus = "PAUSED"
	TaskStatusCompleted TaskStatus = "COMPLETED"
	TaskStatusFailed    TaskStatus = "FAILED"
	TaskStatusSkipped   TaskStatus = "SKIPPED"
	TaskStatusCancelled TaskStatus = "CANCELLED"
	TaskStatusRetrying  TaskStatus = "RETRYING"
)

// NodeType 节点类型
type NodeType string

const (
	NodeTypeHttp        NodeType = "HTTP"
	NodeTypeGrpc        NodeType = "GRPC"
	NodeTypeScript      NodeType = "SCRIPT"
	NodeTypeSubWorkflow NodeType = "SUB_WORKFLOW"
	NodeTypeCondition   NodeType = "CONDITION"
	NodeTypeParallel    NodeType = "PARALLEL"
	NodeTypeSwitch      NodeType = "SWITCH"
	NodeTypeDelay       NodeType = "DELAY"
	NodeTypeCallback    NodeType = "CALLBACK"
	NodeTypeCustom      NodeType = "CUSTOM"

	// 新增通用节点类型
	NodeTypeInput      NodeType = "INPUT"       // 入口节点：传递工作流输入
	NodeTypeOutput     NodeType = "OUTPUT"      // 出口节点：汇聚最终输出
	NodeTypeHumanInput NodeType = "HUMAN_INPUT" // 人类输入：等待外部注入
)

// TriggerType 触发类型
type TriggerType string

const (
	TriggerTypeManual      TriggerType = "MANUAL"
	TriggerTypeSchedule    TriggerType = "SCHEDULE"
	TriggerTypeEvent       TriggerType = "EVENT"
	TriggerTypeApi         TriggerType = "API"
	TriggerTypeSubWorkflow TriggerType = "SUB_WORKFLOW"
)

// Node 状态常量
const (
	RetryModeLinear      = "LINEAR"
	RetryModeExponential = "EXPONENTIAL"
)
