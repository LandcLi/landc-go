package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/LandcLi/landc-go/workflow/pkg/model"
)

// ============================================================
// ExecutionContext — 工作流执行上下文
// 支持：变量传递、事件推送、变量引用解析 ({{nodeID}})
// ============================================================

type ExecutionContext struct {
	mu sync.RWMutex

	Context   context.Context
	Execution *model.Execution
	Workflow  *model.Workflow

	// 节点输出变量，key=nodeID
	Variables map[string]json.RawMessage

	// 节点输出映射的展开数据
	VariableValues map[string]interface{}

	// 事件通道（外部可消费，用于 WebSocket 推送）
	Events chan *model.WorkflowEvent

	// 追踪记录
	Traces []*model.NodeTrace
}

func NewExecutionContext(ctx context.Context, exec *model.Execution, wf *model.Workflow) *ExecutionContext {
	return &ExecutionContext{
		Context:        ctx,
		Execution:      exec,
		Workflow:       wf,
		Variables:      make(map[string]json.RawMessage),
		VariableValues: make(map[string]interface{}),
		Events:         make(chan *model.WorkflowEvent, 256),
	}
}

// SetNodeOutput 设置节点输出
func (ec *ExecutionContext) SetNodeOutput(nodeID string, output json.RawMessage) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.Variables[nodeID] = output

	// 尝试解析为展开数据
	var val interface{}
	if json.Unmarshal(output, &val) == nil {
		ec.VariableValues[nodeID] = val
	}
}

// GetNodeOutput 获取节点原始输出
func (ec *ExecutionContext) GetNodeOutput(nodeID string) json.RawMessage {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.Variables[nodeID]
}

// GetNodeValue 获取节点输出解析后的值
func (ec *ExecutionContext) GetNodeValue(nodeID string) interface{} {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.VariableValues[nodeID]
}

// GetInput 获取节点输入。
// 默认返回所有上游节点输出的合并（key=上游节点ID），
// 当节点配置了 InputMapping 时按映射规则提取指定上游输出。
func (ec *ExecutionContext) GetInput(node *model.Node) json.RawMessage {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	// 优先使用 InputMapping（显式控制）
	if node.InputMapping != nil && len(node.InputMapping) > 0 {
		var mapping map[string]string
		if json.Unmarshal(node.InputMapping, &mapping) == nil {
			result := make(map[string]json.RawMessage)
			for sourceNodeID := range mapping {
				if output, ok := ec.Variables[sourceNodeID]; ok {
					result[sourceNodeID] = output
				}
			}
			data, _ := json.Marshal(result)
			return data
		}
	}

	// 默认：聚合所有上游节点输出
	// 节点执行器可通过 req.Input 拿到上游完整数据
	var allOutputs map[string]json.RawMessage
	hasUpstream := false
	// 查找当前节点在工作流中的上游
	if ec.Workflow != nil {
		for _, edge := range ec.Workflow.Edges {
			if edge.TargetID == node.ID && !edge.Internal {
				if output, ok := ec.Variables[edge.SourceID]; ok {
					if allOutputs == nil {
						allOutputs = make(map[string]json.RawMessage)
					}
					allOutputs[edge.SourceID] = output
					hasUpstream = true
				}
			}
		}
	}

	if hasUpstream {
		data, _ := json.Marshal(allOutputs)
		return data
	}

	// 无上游（根节点），返回执行原始输入
	return ec.Execution.Input
}

// ============================================================
// 变量引用解析 ({{nodeID}} / {{nodeID.path.to.field}})
// ============================================================

var variableRefRE = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// ResolveTemplate 解析模板中的变量引用。
// 支持格式：
//   - {{nodeID}}              — 节点完整输出
//   - {{nodeID.field}}        — 节点输出的 JSON 子字段
//   - {{nodeID.0.field}}      — JSON 数组元素的字段
func (ec *ExecutionContext) ResolveTemplate(template string) string {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	return variableRefRE.ReplaceAllStringFunc(template, func(match string) string {
		path := match[2 : len(match)-2]
		parts := strings.SplitN(path, ".", 2)

		val, ok := ec.VariableValues[parts[0]]
		if !ok {
			// 尝试 raw 方式
			if raw := ec.Variables[parts[0]]; raw != nil {
				return string(raw)
			}
			return match
		}

		if len(parts) > 1 && parts[1] != "" {
			resolved := resolveJSONPath(val, parts[1])
			if resolved != nil {
				return fmt.Sprintf("%v", resolved)
			}
			return match
		}

		return fmt.Sprintf("%v", val)
	})
}

// resolveJSONPath 递归解析 JSON 路径，如 "choices.0.message.content"
func resolveJSONPath(data interface{}, path string) interface{} {
	if path == "" {
		return data
	}
	parts := strings.SplitN(path, ".", 2)

	switch v := data.(type) {
	case map[string]interface{}:
		if next, ok := v[parts[0]]; ok {
			if len(parts) > 1 {
				return resolveJSONPath(next, parts[1])
			}
			return next
		}
	case []interface{}:
		idx := 0
		if _, err := fmt.Sscanf(parts[0], "%d", &idx); err == nil && idx >= 0 && idx < len(v) {
			if len(parts) > 1 {
				return resolveJSONPath(v[idx], parts[1])
			}
			return v[idx]
		}
	}
	return nil
}

// ============================================================
// 事件推送
// ============================================================

func (ec *ExecutionContext) EmitEvent(eventType, nodeID, nodeName, nodeType, content string) {
	evt := model.NewWorkflowEvent(eventType, ec.Execution.ID, nodeID, nodeName, nodeType, content)
	select {
	case ec.Events <- evt:
	default:
	}
}

func (ec *ExecutionContext) EmitErrorEvent(nodeID, nodeName, nodeType, errMsg string) {
	ec.EmitEvent("node.error", nodeID, nodeName, nodeType, errMsg)
}

func (ec *ExecutionContext) EmitCompletedEvent(nodeID, nodeName, nodeType string, duration time.Duration) {
	evt := model.NewWorkflowEvent("node.completed", ec.Execution.ID, nodeID, nodeName, nodeType, "")
	evt.Duration = duration.Milliseconds()
	if output := ec.GetNodeOutput(nodeID); output != nil {
		evt.Output = string(output)
	}
	select {
	case ec.Events <- evt:
	default:
	}
}

// ============================================================
// 追踪记录
// ============================================================

func (ec *ExecutionContext) RecordTrace(nodeID, nodeType, nodeName string, input, output interface{}, duration time.Duration, err error) {
	trace := &model.NodeTrace{
		NodeID:    nodeID,
		NodeType:  nodeType,
		NodeName:  nodeName,
		Input:     input,
		Output:    output,
		Duration:  duration.Milliseconds(),
		Timestamp: time.Now(),
	}
	if err != nil {
		trace.Error = err.Error()
	}
	ec.mu.Lock()
	ec.Traces = append(ec.Traces, trace)
	ec.mu.Unlock()
}

// Close 关闭事件通道
func (ec *ExecutionContext) Close() {
	close(ec.Events)
}
