package executor

import (
	"sync"

	"github.com/LandcLi/landc-go/workflow/pkg/model"
)

// Registry 执行器注册表
type Registry struct {
	mu        sync.RWMutex
	executors map[string]NodeExecutor
}

// NewRegistry 创建执行器注册表
func NewRegistry() *Registry {
	return &Registry{
		executors: make(map[string]NodeExecutor),
	}
}

// Register 注册执行器
func (r *Registry) Register(executor NodeExecutor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executors[executor.Type()] = executor
}

// Get 获取执行器
func (r *Registry) Get(nodeType model.NodeType) NodeExecutor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	exec, ok := r.executors[string(nodeType)]
	if !ok {
		return nil
	}
	return exec
}

// GetByTypeStr 根据字符串类型获取
func (r *Registry) GetByTypeStr(nodeType string) NodeExecutor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.executors[nodeType]
}

// RegisterDefault 注册所有默认执行器
func RegisterDefault(reg *Registry) {
	reg.Register(NewHTTPExecutor())
	reg.Register(NewScriptExecutor())
	reg.Register(NewSubWorkflowExecutor())
	reg.Register(NewDelayExecutor())
	reg.Register(NewInputNodeExecutor())
	reg.Register(NewOutputNodeExecutor())
	reg.Register(NewConditionNodeExecutor())
	reg.Register(NewSwitchNodeExecutor())
	// HumanInputExecutor 需要通过 Engine 注入 waitingInputs, 由 Engine 单独注册
}

// RegisterHumanInput 注册人类输入执行器（需要 Engine 提供 channel 工厂）
func RegisterHumanInput(reg *Registry, waitingInputs func(execID, nodeID string) <-chan string) {
	reg.Register(NewHumanInputExecutor(waitingInputs))
}
