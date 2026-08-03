package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LandcLi/landc-go/workflow/pkg/model"
)

// ============================================================
// runningExecution — 运行时状态（包含信号通道）
// ============================================================

type runningExecution struct {
	cancel  context.CancelFunc
	started time.Time

	// 事件通道（外部通过 Events() 获取）
	events chan *model.WorkflowEvent

	// 人类输入
	waitingInputs map[string]chan string

	// 调试模式
	debugMode   bool
	breakpoints map[string]bool
	stepCh      chan struct{}

	// 暂停/恢复信号（设计文档中的 SignalService 思路）
	// PauseWorkflow 关闭此 channel → executeDAG 立即感知
	pausedCh chan struct{}
	// 暂停时保存的执行快照，resume 时重建
	pausedAt time.Time
	snapshot *dagSnapshot
}

// dagSnapshot DAG 执行快照（暂停时保存，恢复时重建）
type dagSnapshot struct {
	CompletedNodes map[string]bool            `json:"completed_nodes"`
	Variables      map[string]json.RawMessage `json:"variables"`
	InDegree       map[string]int             `json:"in_degree"`
	Ready          []string                   `json:"ready"`
}

// ============================================================
// 1. 信号通道管理（替代 DB 轮询）
// ============================================================

// initPauseSignal 初始化暂停信号通道
func (e *Engine) initPauseSignal(execID string) {
	e.runningMu.Lock()
	defer e.runningMu.Unlock()
	if r, ok := e.running[execID]; ok {
		if r.pausedCh == nil {
			r.pausedCh = make(chan struct{})
		}
	}
}

// signalPause 发送暂停信号（关闭 channel，所有监听者立刻感知）
func (e *Engine) signalPause(execID string) {
	e.runningMu.Lock()
	defer e.runningMu.Unlock()
	if r, ok := e.running[execID]; ok {
		select {
		case <-r.pausedCh:
			// 已暂停，忽略
		default:
			close(r.pausedCh)
			r.pausedAt = time.Now()
		}
	}
}

// signalResume 重置暂停信号（重新创建 channel）
func (e *Engine) signalResume(execID string) {
	e.runningMu.Lock()
	defer e.runningMu.Unlock()
	if r, ok := e.running[execID]; ok {
		r.pausedCh = make(chan struct{})
		r.pausedAt = time.Time{}
		r.snapshot = nil
	}
}

// saveDAGSnapshot 保存 DAG 快照到 runningExecution
func (e *Engine) saveDAGSnapshot(execID string, completed map[string]bool, vars map[string]json.RawMessage, inDegree map[string]int, ready []string) {
	snap := &dagSnapshot{
		CompletedNodes: completed,
		Variables:      vars,
		InDegree:       inDegree,
		Ready:          ready,
	}
	// 同时持久化到 DB（用于进程重启恢复）
	snapJSON, _ := json.Marshal(snap)
	_ = snapJSON // 实际使用需存到 Execution.StateData

	e.runningMu.Lock()
	if r, ok := e.running[execID]; ok {
		r.snapshot = snap
	}
	e.runningMu.Unlock()
}

// ============================================================
// 2. 人类输入 (HumanInput)
// ============================================================

func (e *Engine) waitForHumanInput(execID, nodeID string) <-chan string {
	e.runningMu.Lock()
	defer e.runningMu.Unlock()

	running, ok := e.running[execID]
	if !ok {
		ch := make(chan string, 1)
		close(ch)
		return ch
	}
	if running.waitingInputs == nil {
		running.waitingInputs = make(map[string]chan string)
	}
	ch := make(chan string, 1)
	running.waitingInputs[nodeID] = ch
	return ch
}

func (e *Engine) ProvideHumanInput(ctx context.Context, execID, nodeID, input string) error {
	e.runningMu.RLock()
	running, ok := e.running[execID]
	e.runningMu.RUnlock()
	if !ok {
		return fmt.Errorf("execution %s not found", execID)
	}

	e.runningMu.Lock()
	ch, ok := running.waitingInputs[nodeID]
	e.runningMu.Unlock()
	if !ok {
		return fmt.Errorf("no waiting human input for execution %s node %s", execID, nodeID)
	}

	select {
	case ch <- input:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("human input channel full or closed")
	}
}

// ============================================================
// 3. 事件订阅
// ============================================================

func (e *Engine) Events(execID string) (<-chan *model.WorkflowEvent, error) {
	e.runningMu.RLock()
	running, ok := e.running[execID]
	e.runningMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("execution %s not found", execID)
	}

	e.runningMu.Lock()
	defer e.runningMu.Unlock()
	if running.events == nil {
		running.events = make(chan *model.WorkflowEvent, 256)
	}
	return running.events, nil
}

func (e *Engine) eventsForExec(execID string) chan *model.WorkflowEvent {
	e.runningMu.RLock()
	running, ok := e.running[execID]
	e.runningMu.RUnlock()
	if !ok {
		return nil
	}
	e.runningMu.Lock()
	defer e.runningMu.Unlock()
	if running.events == nil {
		running.events = make(chan *model.WorkflowEvent, 256)
	}
	return running.events
}

// ============================================================
// 4. 调试模式
// ============================================================

func (e *Engine) SetBreakpoints(ctx context.Context, execID string, nodeIDs []string) error {
	e.runningMu.RLock()
	running, ok := e.running[execID]
	e.runningMu.RUnlock()
	if !ok {
		return fmt.Errorf("execution %s not found", execID)
	}

	e.runningMu.Lock()
	defer e.runningMu.Unlock()
	if running.breakpoints == nil {
		running.breakpoints = make(map[string]bool)
	}
	for _, id := range nodeIDs {
		running.breakpoints[id] = true
	}
	running.debugMode = true
	return nil
}

func (e *Engine) StepNext(ctx context.Context, execID string) error {
	e.runningMu.RLock()
	running, ok := e.running[execID]
	e.runningMu.RUnlock()
	if !ok {
		return fmt.Errorf("execution %s not found", execID)
	}

	e.runningMu.Lock()
	defer e.runningMu.Unlock()
	if running.stepCh != nil {
		select {
		case running.stepCh <- struct{}{}:
		default:
		}
	}
	return nil
}

func (e *Engine) shouldPauseForDebug(execID, nodeID string) bool {
	e.runningMu.RLock()
	running, ok := e.running[execID]
	e.runningMu.RUnlock()
	return ok && running.debugMode && running.breakpoints[nodeID]
}

func (e *Engine) waitForDebugStep(execID string) {
	e.runningMu.Lock()
	running, ok := e.running[execID]
	if !ok {
		e.runningMu.Unlock()
		return
	}
	if running.stepCh == nil {
		running.stepCh = make(chan struct{}, 1)
	}
	stepCh := running.stepCh
	e.runningMu.Unlock()
	<-stepCh
}
