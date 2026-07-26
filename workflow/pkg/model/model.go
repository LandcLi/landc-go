package model

import (
	"encoding/json"
	"time"
)

// ============================================================
// Workflow 工作流定义
// ============================================================

type Workflow struct {
	ID          string         `gorm:"primaryKey;size:64;comment:工作流ID" json:"id"`
	Name        string         `gorm:"size:256;not null;index;comment:工作流名称" json:"name"`
	Description string         `gorm:"size:1024;comment:描述" json:"description"`
	Version     int            `gorm:"not null;default:1;comment:版本号" json:"version"`
	Status      WorkflowStatus `gorm:"size:32;not null;default:ACTIVE;index;comment:状态" json:"status"`
	Timeout     int64          `gorm:"comment:超时时间(秒),0表示不超时" json:"timeout"`
	MaxRetries  int            `gorm:"not null;default:0;comment:全局最大重试次数" json:"max_retries"`
	Nodes       []*Node        `gorm:"foreignKey:WorkflowID;constraint:OnDelete:CASCADE;" json:"nodes,omitempty"`
	Edges       []*Edge        `gorm:"foreignKey:WorkflowID;constraint:OnDelete:CASCADE;" json:"edges,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (w *Workflow) TableName() string { return "wf_workflows" }

func (w *Workflow) IsValid() bool {
	if w.Name == "" || len(w.Nodes) == 0 {
		return false
	}
	seen := make(map[string]bool)
	for _, n := range w.Nodes {
		if n.ID == "" || seen[n.ID] {
			return false
		}
		seen[n.ID] = true
	}
	return true
}

// ============================================================
// Node 工作流节点定义
// ============================================================

type Node struct {
	ID          string   `gorm:"primaryKey;size:64;comment:节点ID" json:"id"`
	WorkflowID  string   `gorm:"size:64;not null;index;comment:所属工作流ID" json:"workflow_id"`
	Name        string   `gorm:"size:256;not null;comment:节点名称" json:"name"`
	Type        NodeType `gorm:"size:32;not null;comment:节点类型" json:"type"`
	Description string   `gorm:"size:1024;comment:描述" json:"description,omitempty"`

	// 超时和重试
	Timeout    int64 `gorm:"comment:节点超时(秒)" json:"timeout"`
	MaxRetries int   `gorm:"not null;default:0;comment:重试次数" json:"max_retries"`

	// 重试策略
	RetryDelay    int64  `gorm:"not null;default:0;comment:重试间隔(秒)" json:"retry_delay"`
	RetryMode     string `gorm:"size:32;not null;default:LINEAR;comment:重试模式: LINEAR/EXPONENTIAL" json:"retry_mode"`
	RetryMaxDelay int64  `gorm:"not null;default:300;comment:最大重试延迟(秒)" json:"retry_max_delay"`

	// 跳过策略：节点失败时是否跳过继续执行下游
	SkipOnFailure bool `gorm:"not null;default:false;comment:失败时跳过" json:"skip_on_failure"`

	// 节点配置（JSON 格式，根据 Type 不同而不同）
	Config json.RawMessage `gorm:"type:text;comment:节点配置(JSON)" json:"config,omitempty"`

	// 输入映射
	InputMapping  json.RawMessage `gorm:"type:text;comment:输入映射(JSON)" json:"input_mapping,omitempty"`
	OutputMapping json.RawMessage `gorm:"type:text;comment:输出映射(JSON)" json:"output_mapping,omitempty"`

	// 条件表达式（CONDITION/SWITCH 节点使用）
	ConditionExpr string `gorm:"size:1024;comment:条件表达式" json:"condition_expr,omitempty"`

	// 并行分支配置（PARALLEL 节点使用）
	ParallelBranches int `gorm:"not null;default:1;comment:并行分支数" json:"parallel_branches"`

	OrderNo   int       `gorm:"not null;default:0;comment:排序号" json:"order_no"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (n *Node) TableName() string { return "wf_nodes" }

// ============================================================
// Edge DAG 边定义（支持端口级条件分支）
// ============================================================

type Edge struct {
	ID         string `gorm:"primaryKey;size:64;comment:边ID" json:"id"`
	WorkflowID string `gorm:"size:64;not null;index;comment:所属工作流ID" json:"workflow_id"`
	SourceID   string `gorm:"size:64;not null;index;comment:上游节点ID" json:"source_id"`
	TargetID   string `gorm:"size:64;not null;index;comment:下游节点ID" json:"target_id"`
	// 源端口（条件分支用，如 "true"/"false"/"default"/"approved"/"rejected"）
	SourcePort string `gorm:"size:64;comment:源端口(条件分支)" json:"source_port,omitempty"`
	// 目标端口（如 "input"/"loop_target"）
	TargetPort string `gorm:"size:64;comment:目标端口" json:"target_port,omitempty"`
	// 条件表达式（动态分支，优先级低于 SourcePort）
	ConditionExpr string `gorm:"size:1024;comment:条件表达式" json:"condition_expr,omitempty"`
	// 标签（同 source_port 语义，兼容前端面板显示）
	Label     string    `gorm:"size:64;comment:边标签" json:"label,omitempty"`
	OrderNo   int       `gorm:"not null;default:0;comment:排序号" json:"order_no"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (e *Edge) TableName() string { return "wf_edges" }

// ============================================================
// Execution 工作流执行实例
// ============================================================

type Execution struct {
	ID           string          `gorm:"primaryKey;size:64;comment:执行ID" json:"id"`
	WorkflowID   string          `gorm:"size:64;not null;index;comment:工作流ID" json:"workflow_id"`
	WorkflowName string          `gorm:"size:256;comment:快照:工作流名称" json:"workflow_name"`
	WorkflowVer  int             `gorm:"comment:快照:工作流版本" json:"workflow_ver"`
	TriggerType  TriggerType     `gorm:"size:32;not null;default:API;comment:触发类型" json:"trigger_type"`
	TriggerID    string          `gorm:"size:128;index;comment:外部触发ID(用于幂等)" json:"trigger_id"`
	Status       ExecutionStatus `gorm:"size:32;not null;default:PENDING;index;comment:执行状态" json:"status"`

	Input     json.RawMessage `gorm:"type:longtext;comment:输入数据" json:"input,omitempty"`
	Output    json.RawMessage `gorm:"type:longtext;comment:输出数据" json:"output,omitempty"`
	StateData json.RawMessage `gorm:"type:longtext;comment:状态快照(用于暂停恢复)" json:"state_data,omitempty"`

	CurrentNodeID string `gorm:"size:64;comment:当前节点ID(可重入恢复点)" json:"current_node_id"`

	Error string `gorm:"type:text;comment:错误信息" json:"error,omitempty"`

	Timeout   int64      `gorm:"comment:超时时间(秒)" json:"timeout"`
	ExpiresAt *time.Time `gorm:"index;comment:超时截止时间" json:"expires_at,omitempty"`

	Version   int        `gorm:"not null;default:1;comment:乐观锁版本" json:"version"`
	StartedAt *time.Time `gorm:"comment:开始时间" json:"started_at,omitempty"`
	FinishedAt *time.Time `gorm:"comment:完成时间" json:"finished_at,omitempty"`
	CreatedAt  time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (e *Execution) TableName() string { return "wf_executions" }

func (e *Execution) IsFinal() bool {
	switch e.Status {
	case ExecutionStatusCompleted, ExecutionStatusFailed, ExecutionStatusCancelled, ExecutionStatusTimeout:
		return true
	}
	return false
}

func (e *Execution) CanTransition(target ExecutionStatus) bool {
	if e.IsFinal() {
		return false
	}
	switch e.Status {
	case ExecutionStatusPending:
		return target == ExecutionStatusRunning || target == ExecutionStatusCancelled
	case ExecutionStatusRunning:
		return target == ExecutionStatusCompleted || target == ExecutionStatusFailed ||
			target == ExecutionStatusPaused || target == ExecutionStatusCancelled ||
			target == ExecutionStatusTimeout
	case ExecutionStatusPaused:
		return target == ExecutionStatusRunning || target == ExecutionStatusCancelled
	}
	return false
}

// ============================================================
// Task 节点执行实例
// ============================================================

type Task struct {
	ID           string     `gorm:"primaryKey;size:64;comment:任务ID" json:"id"`
	ExecutionID  string     `gorm:"size:64;not null;index;comment:执行ID" json:"execution_id"`
	NodeID       string     `gorm:"size:64;not null;index;comment:节点ID" json:"node_id"`
	NodeName     string     `gorm:"size:256;comment:快照:节点名称" json:"node_name"`
	NodeType     NodeType   `gorm:"size:32;comment:快照:节点类型" json:"node_type"`
	Status       TaskStatus `gorm:"size:32;not null;default:PENDING;index;comment:任务状态" json:"status"`

	Input  json.RawMessage `gorm:"type:longtext;comment:输入数据" json:"input,omitempty"`
	Output json.RawMessage `gorm:"type:longtext;comment:输出数据" json:"output,omitempty"`
	Error  string          `gorm:"type:text;comment:错误信息" json:"error,omitempty"`

	RetryCount int  `gorm:"not null;default:0;comment:已重试次数" json:"retry_count"`
	MaxRetries int  `gorm:"not null;default:0;comment:最大重试次数" json:"max_retries"`
	IsRetry    bool `gorm:"not null;default:false;comment:是否重试执行" json:"is_retry"`

	AttemptID string `gorm:"size:128;index;comment:执行尝试ID(用于幂等)" json:"attempt_id"`

	WorkerID    string     `gorm:"size:128;index;comment:处理该任务的WorkerID" json:"worker_id"`
	ScheduledAt *time.Time `gorm:"index;comment:计划执行时间" json:"scheduled_at,omitempty"`
	StartedAt   *time.Time `gorm:"comment:开始时间" json:"started_at,omitempty"`
	FinishedAt  *time.Time `gorm:"comment:完成时间" json:"finished_at,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (t *Task) TableName() string { return "wf_tasks" }

func (t *Task) IsFinal() bool {
	switch t.Status {
	case TaskStatusCompleted, TaskStatusFailed, TaskStatusSkipped, TaskStatusCancelled:
		return true
	}
	return false
}

// ============================================================
// Worker 工作节点注册信息
// ============================================================

type Worker struct {
	ID        string    `gorm:"primaryKey;size:128;comment:WorkerID" json:"id"`
	Address   string    `gorm:"size:256;not null;comment:Worker地址" json:"address"`
	Group     string    `gorm:"size:64;index;comment:Worker分组" json:"group"`
	Status    string    `gorm:"size:32;not null;default:ACTIVE;comment:状态" json:"status"`
	Tags      string    `gorm:"size:512;comment:标签(JSON数组)" json:"tags"`
	Heartbeat time.Time `gorm:"index;comment:最近心跳时间" json:"heartbeat"`
	StartedAt time.Time `gorm:"comment:启动时间" json:"started_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (w *Worker) TableName() string { return "wf_workers" }
