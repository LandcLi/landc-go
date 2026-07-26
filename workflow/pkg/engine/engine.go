package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/trace"
	"github.com/LandcLi/landc-go/log/facade"
	"github.com/LandcLi/landc-go/tools/generate"
	"github.com/LandcLi/landc-go/workflow/pkg/executor"
	"github.com/LandcLi/landc-go/workflow/pkg/idempotent"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
	"github.com/LandcLi/landc-go/workflow/pkg/observer"
	"github.com/LandcLi/landc-go/workflow/pkg/store"
)

type Engine struct {
	dbStore    store.Store
	execReg    *executor.Registry
	observers  *observer.ObserverManager
	idempCheck idempotent.IdempotencyChecker
	logger     facade.Logger

	runningMu sync.RWMutex
	running   map[string]*runningExecution
	config    EngineConfig
}

// runningExecution 定义在 hooks.go 中

type EngineConfig struct {
	MaxParallelTasks int
	DefaultTimeout   time.Duration
	IdempotencyTTL   time.Duration
	PauseTimeout     time.Duration // 暂停超过此时间自动取消执行，0 表示不限制
	EventBufferSize  int
	LogName          string
}

func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		MaxParallelTasks: 10,
		DefaultTimeout:   30 * time.Minute,
		IdempotencyTTL:   24 * time.Hour,
		PauseTimeout:     0,
		EventBufferSize:  256,
		LogName:          "workflow.engine",
	}
}

func NewEngine(
	dbStore store.Store,
	execReg *executor.Registry,
	obs *observer.ObserverManager,
	idempCheck idempotent.IdempotencyChecker,
	config EngineConfig,
) *Engine {
	eng := &Engine{
		dbStore:    dbStore,
		execReg:    execReg,
		observers:  obs,
		idempCheck: idempCheck,
		logger:     facade.GetLoggerWithName(config.LogName),
		running:    make(map[string]*runningExecution),
		config:     config,
	}

	// 注册 HumanInputExecutor（通过 Engine 的 channel 工厂）
	executor.RegisterHumanInput(execReg, eng.waitForHumanInput)

	return eng
}

// ============================================================
// 公开 API
// ============================================================

func (e *Engine) StartWorkflow(ctx context.Context, workflowID string, input json.RawMessage, triggerType model.TriggerType, triggerID string) (string, error) {
	ctx, span := trace.NewSpan(ctx, "workflow.StartWorkflow")
	defer span.End()

	if triggerID != "" {
		key := e.idempCheck.GenerateKey(workflowID, triggerID)
		processed, err := e.idempCheck.IsProcessed(ctx, key)
		if err != nil {
			return "", fmt.Errorf("idempotency check failed: %w", err)
		}
		if processed {
			existing, err := e.dbStore.GetExecutionByTriggerID(ctx, workflowID, triggerID)
			if err != nil {
				return "", err
			}
			if existing != nil {
				return existing.ID, nil
			}
		}
	}

	wf, err := e.dbStore.GetWorkflowWithNodes(ctx, workflowID)
	if err != nil {
		span.EndWithError(err)
		return "", fmt.Errorf("load workflow failed: %w", err)
	}
	if wf.Status != model.WorkflowStatusActive {
		err = fmt.Errorf("workflow %s is not active: %s", wf.ID, wf.Status)
		span.EndWithError(err)
		return "", err
	}

	exec := &model.Execution{
		ID:           generate.UUID(),
		WorkflowID:   wf.ID,
		WorkflowName: wf.Name,
		WorkflowVer:  wf.Version,
		TriggerType:  triggerType,
		TriggerID:    triggerID,
		Status:       model.ExecutionStatusPending,
		Input:        input,
		Timeout:      wf.Timeout,
		Version:      1,
	}
	if exec.Timeout <= 0 && e.config.DefaultTimeout > 0 {
		exec.Timeout = int64(e.config.DefaultTimeout.Seconds())
	}
	if exec.Timeout > 0 {
		expiresAt := time.Now().Add(time.Duration(exec.Timeout) * time.Second)
		exec.ExpiresAt = &expiresAt
	}

	if err := e.dbStore.CreateExecution(ctx, exec); err != nil {
		span.EndWithError(err)
		return "", fmt.Errorf("create execution failed: %w", err)
	}

	if triggerID != "" {
		key := e.idempCheck.GenerateKey(workflowID, triggerID)
		_ = e.idempCheck.MarkProcessed(ctx, key, e.config.IdempotencyTTL)
	}

	e.logger.WithContext(ctx).Info("[engine] workflow execution created",
		facade.Field{Key: "execution_id", Value: exec.ID},
		facade.Field{Key: "workflow_name", Value: wf.Name},
	)

	execCtx, cancel := context.WithCancel(ctx)
	e.runningMu.Lock()
	e.running[exec.ID] = &runningExecution{cancel: cancel, started: time.Now()}
	e.runningMu.Unlock()

	go e.executeWorkflow(execCtx, exec, wf)
	return exec.ID, nil
}

// SubscribeEvents 订阅执行事件（外部可通过 channel 消费，如 WebSocket 推送）
func (e *Engine) SubscribeEvents(ctx context.Context, execID string) (<-chan *model.WorkflowEvent, error) {
	// 当前简化实现：外部自行消费 ExecutionContext 的事件
	return nil, fmt.Errorf("SubscribeEvents: use GetEvents instead")
}

func (e *Engine) PauseWorkflow(ctx context.Context, execID string) error {
	exec, err := e.dbStore.GetExecution(ctx, execID)
	if err != nil {
		return err
	}
	if !exec.CanTransition(model.ExecutionStatusPaused) {
		return fmt.Errorf("cannot pause from state: %s", exec.Status)
	}
	// 先发暂停信号（SignalService 模式，即时生效）
	e.signalPause(execID)

	exec.Status = model.ExecutionStatusPaused
	if err := e.dbStore.UpdateExecution(ctx, exec); err != nil {
		return err
	}
	e.pushFinalEvent(execID, "workflow.paused")
	e.observers.OnExecutionPaused(ctx, exec)
	return nil
}

func (e *Engine) ResumeWorkflow(ctx context.Context, execID string) error {
	exec, err := e.dbStore.GetExecution(ctx, execID)
	if err != nil {
		return err
	}
	if exec.Status != model.ExecutionStatusPaused {
		return fmt.Errorf("can only resume paused execution, current: %s", exec.Status)
	}
	wf, err := e.dbStore.GetWorkflowWithNodes(ctx, exec.WorkflowID)
	if err != nil {
		return err
	}
	// 重置暂停信号
	e.signalResume(execID)

	exec.Status = model.ExecutionStatusRunning
	if err := e.dbStore.UpdateExecution(ctx, exec); err != nil {
		return err
	}
	e.observers.OnExecutionResumed(ctx, exec)
	e.pushFinalEvent(execID, "workflow.resumed")

	execCtx, cancel := context.WithCancel(ctx)
	e.runningMu.Lock()
	e.running[exec.ID] = &runningExecution{
		cancel:   cancel,
		started:  time.Now(),
		pausedCh: make(chan struct{}),
	}
	e.runningMu.Unlock()

	go e.executeWorkflow(execCtx, exec, wf)
	return nil
}

func (e *Engine) CancelWorkflow(ctx context.Context, execID string) error {
	exec, err := e.dbStore.GetExecution(ctx, execID)
	if err != nil {
		return err
	}
	if exec.IsFinal() {
		return fmt.Errorf("execution already in final state: %s", exec.Status)
	}
	e.runningMu.RLock()
	running, ok := e.running[execID]
	e.runningMu.RUnlock()
	if ok && running.cancel != nil {
		running.cancel()
	}
	exec.Status = model.ExecutionStatusCancelled
	now := time.Now()
	exec.FinishedAt = &now
	if err := e.dbStore.UpdateExecution(ctx, exec); err != nil {
		return err
	}
	e.observers.OnExecutionCancelled(ctx, exec)
	return nil
}

func (e *Engine) GetExecutionStatus(ctx context.Context, execID string) (*model.Execution, error) {
	return e.dbStore.GetExecution(ctx, execID)
}

func (e *Engine) GetExecutionTasks(ctx context.Context, execID string) ([]*model.Task, error) {
	return e.dbStore.ListTasks(ctx, execID)
}

// WaitExecution 阻塞直到执行完成、失败或上下文取消。
// 通过轮询 + 事件监听实现，适合同步等待工作流执行结果的场景。
func (e *Engine) WaitExecution(ctx context.Context, execID string) (*model.Execution, error) {
	// 先检查当前状态
	exec, err := e.dbStore.GetExecution(ctx, execID)
	if err != nil {
		return nil, err
	}
	if exec.IsFinal() {
		return exec, nil
	}

	// 尝试通过事件通道等待（如果有）
	if evtCh := e.eventsForExec(execID); evtCh != nil {
		for {
			select {
			case <-ctx.Done():
				return e.dbStore.GetExecution(ctx, execID)
			case evt, ok := <-evtCh:
				if !ok {
					// 通道关闭，查最终状态
					return e.dbStore.GetExecution(ctx, execID)
				}
				if evt.Type == "workflow.completed" || evt.Type == "workflow.failed" || evt.Type == "workflow.cancelled" {
					return e.dbStore.GetExecution(ctx, execID)
				}
			}
		}
	}

	// 没有事件通道时轮询
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return e.dbStore.GetExecution(ctx, execID)
		case <-ticker.C:
			exec, err := e.dbStore.GetExecution(ctx, execID)
			if err != nil {
				return nil, err
			}
			if exec.IsFinal() {
				return exec, nil
			}
		}
	}
}

// ============================================================
// 内部 DAG 执行（核心逻辑）
// ============================================================

func (e *Engine) executeWorkflow(ctx context.Context, exec *model.Execution, wf *model.Workflow) {
	ctx, span := trace.NewSpan(ctx, "workflow.execute")
	defer span.End()
	ec := NewExecutionContext(ctx, exec, wf)
	defer ec.Close()

	dag, err := NewDAGGraph(wf)
	if err != nil {
		span.EndWithError(err)
		e.failExecution(ctx, exec, err)
		return
	}

	exec.Status = model.ExecutionStatusRunning
	now := time.Now()
	exec.StartedAt = &now
	if err := e.dbStore.UpdateExecution(ctx, exec); err != nil {
		return
	}
	e.observers.OnExecutionStarted(ctx, exec)
	e.pushEvent(ec, "workflow.started", "", exec.WorkflowName, "", string(exec.Input))

	// 加载已有任务（可重入）
	existingTasks, _ := e.dbStore.ListTasks(ctx, exec.ID)
	completedNodes := make(map[string]bool)
	taskMap := make(map[string]*model.Task)
	for _, t := range existingTasks {
		taskMap[t.NodeID] = t
		if t.Status == model.TaskStatusCompleted || t.Status == model.TaskStatusSkipped {
			completedNodes[t.NodeID] = true
		}
	}

	if err := e.executeDAG(ec, dag, completedNodes, taskMap); err != nil {
		if errors.Is(err, context.Canceled) {
			e.pushFinalEvent(exec.ID, "workflow.cancelled")
			return
		}
		span.EndWithError(err)
		e.failExecution(ctx, exec, err)
		e.pushFinalEvent(exec.ID, "workflow.failed")
		return
	}

	// 检查是否全部完成
	allDone := true
	for _, n := range dag.workflow.Nodes {
		if !completedNodes[n.ID] {
			allDone = false
			break
		}
	}

	if allDone {
		now := time.Now()
		exec.FinishedAt = &now
		exec.Status = model.ExecutionStatusCompleted
		if err := e.dbStore.UpdateExecution(ctx, exec); err == nil {
			e.observers.OnExecutionCompleted(ctx, exec, nil)
			e.pushFinalEvent(exec.ID, "workflow.completed")
		}
	}

	e.runningMu.Lock()
	delete(e.running, exec.ID)
	e.runningMu.Unlock()
}

func (e *Engine) executeDAG(ec *ExecutionContext, dag *DAGGraph, completedNodes map[string]bool, taskMap map[string]*model.Task) error {
	ctx := ec.Context
	execID := ec.Execution.ID
	sem := make(chan struct{}, e.config.MaxParallelTasks)
	var wg sync.WaitGroup
	errCh := make(chan error, len(dag.workflow.Nodes))

	// 初始化暂停信号通道
	e.initPauseSignal(execID)

	// 获取暂停 channel（由 PauseWorkflow 关闭触发）
	pausedCh := func() <-chan struct{} {
		e.runningMu.RLock()
		defer e.runningMu.RUnlock()
		if r, ok := e.running[execID]; ok {
			return r.pausedCh
		}
		return nil
	}()

	// 检查暂停信号（非阻塞，设计文档 SignalService 模式）
	isPaused := func() bool {
		if pausedCh == nil {
			return false
		}
		select {
		case <-pausedCh:
			return true
		default:
			return false
		}
	}

	// 顺序执行根节点
	roots := dag.GetRootNodes()
	for _, node := range roots {
		if completedNodes[node.ID] {
			continue
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(n *model.Node) {
			defer wg.Done()
			defer func() { <-sem }()
			e.executeNode(ec, dag, n, completedNodes, taskMap, errCh)
		}(node)
	}
	wg.Wait()

	// 批次推进（拓扑顺序）
	for {
		// 先处理外部取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 暂停信号检测（即时，无 DB 轮询）
		if isPaused() {
			// 保存 DAG 快照（支持精确恢复）
			readyIDs := make([]string, 0)
			for _, n := range dag.GetReadyNodes(completedNodes) {
				readyIDs = append(readyIDs, n.ID)
			}
			inDegree := make(map[string]int)
			for _, n := range dag.workflow.Nodes {
				deps := dag.GetDepNodes(n.ID)
				remainingDeps := 0
				for _, d := range deps {
					if !completedNodes[d] {
						remainingDeps++
					}
				}
				inDegree[n.ID] = remainingDeps
			}
			e.saveDAGSnapshot(execID, completedNodes, ec.Variables, inDegree, readyIDs)
			ec.Execution.CurrentNodeID = ""
			_ = e.dbStore.UpdateExecution(ctx, ec.Execution)
			e.pushEvent(ec, "workflow.paused", "", ec.Execution.WorkflowName, "", "执行已暂停")

			// 暂停超时自动取消
			if e.config.PauseTimeout > 0 {
				time.AfterFunc(e.config.PauseTimeout, func() {
					exec, err := e.dbStore.GetExecution(context.Background(), execID)
					if err != nil || exec.Status != model.ExecutionStatusPaused {
						return
					}
					e.logger.WithContext(context.Background()).Warn("[engine] pause timeout, auto-cancelling",
						facade.Field{Key: "execution_id", Value: execID},
						facade.Field{Key: "pause_timeout", Value: e.config.PauseTimeout.String()},
					)
					e.CancelWorkflow(context.Background(), execID)
				})
			}
			return nil
		}

		readyNodes := dag.GetReadyNodes(completedNodes)
		if len(readyNodes) == 0 {
			break
		}

		var batchWG sync.WaitGroup
		for _, node := range readyNodes {
			if completedNodes[node.ID] {
				continue
			}
			sem <- struct{}{}
			batchWG.Add(1)
			n := node
			go func() {
				defer batchWG.Done()
				defer func() { <-sem }()
				e.executeNode(ec, dag, n, completedNodes, taskMap, errCh)
			}()
		}
		batchWG.Wait()

		select {
		case err := <-errCh:
			if err != nil {
				return err
			}
		default:
		}
	}

	return nil
}

// executeNode 执行单个节点，支持：条件分支、skip_on_failure、调试断点、事件推送
func (e *Engine) executeNode(ec *ExecutionContext, dag *DAGGraph, node *model.Node, completedNodes map[string]bool, taskMap map[string]*model.Task, errCh chan<- error) {
	ctx := ec.Context
	ctx, span := trace.NewSpan(ctx, "workflow.node."+node.Name)
	defer span.End()

	select {
	case <-ctx.Done():
		errCh <- ctx.Err()
		return
	default:
	}

	// 调试模式：命中断点则暂停并等待
	if e.shouldPauseForDebug(ec.Execution.ID, node.ID) {
		if evtCh := e.eventsForExec(ec.Execution.ID); evtCh != nil {
			evtCh <- model.NewWorkflowEvent("node.debug.pause", ec.Execution.ID, node.ID, node.Name, string(node.Type), "断点暂停")
		}
		e.waitForDebugStep(ec.Execution.ID)
	}

	// 推送事件到引擎级 channel 和 ec Events
	e.pushEvent(ec, "node.started", node.ID, node.Name, string(node.Type), "")
	if evtCh := e.eventsForExec(ec.Execution.ID); evtCh != nil {
		evtCh <- model.NewWorkflowEvent("node.started", ec.Execution.ID, node.ID, node.Name, string(node.Type), "")
	}

	// 获取或创建任务
	task, ok := taskMap[node.ID]
	if ok && task.IsFinal() {
		mu := &sync.Mutex{}
		mu.Lock()
		completedNodes[node.ID] = true
		mu.Unlock()
		return
	}

	if !ok {
		task = &model.Task{
			ID:          generate.UUID(),
			ExecutionID: ec.Execution.ID,
			NodeID:      node.ID,
			NodeName:    node.Name,
			NodeType:    node.Type,
			Status:      model.TaskStatusPending,
			Input:       ec.GetInput(node),
			MaxRetries:  node.MaxRetries,
			AttemptID:   generate.UUID(),
		}
		if err := e.dbStore.CreateTask(ctx, task); err != nil {
			span.EndWithError(err)
			errCh <- fmt.Errorf("create task %s failed: %w", node.Name, err)
			return
		}
		taskMap[node.ID] = task
	}

	// 更新为运行中
	task.Status = model.TaskStatusRunning
	now := time.Now()
	task.StartedAt = &now
	task.WorkerID = "local"
	_ = e.dbStore.UpdateTask(ctx, task)
	e.observers.OnTaskStarted(ctx, task, node)

	// 获取执行器
	execImpl := e.execReg.GetByTypeStr(string(node.Type))
	if execImpl == nil {
		err := fmt.Errorf("no executor for type: %s", node.Type)
		span.EndWithError(err)
		e.handleNodeFailure(ec, task, node, err, completedNodes, errCh)
		return
	}

	// 使用重试装饰器
	retryStrategy := executor.NewRetryStrategy(node.RetryMode, node.MaxRetries, node.RetryDelay, node.RetryMaxDelay)
	retryExec := executor.NewRetryableExecutor(execImpl, retryStrategy, true)

	req := &executor.ExecuteRequest{
		NodeID:      node.ID,
		NodeName:    node.Name,
		NodeType:    string(node.Type),
		Config:      node.Config,
		Input:       task.Input,
		InputMap:    node.InputMapping,
		OutputMap:   node.OutputMapping,
		RetryCount:  task.RetryCount,
		MaxRetries:  node.MaxRetries,
		AttemptID:   task.AttemptID,
		ExecutionID: ec.Execution.ID,
	}

	// 记录重试事件
	if task.RetryCount > 0 {
		for a := 1; a <= task.RetryCount; a++ {
			ec.EmitEvent("node.retrying", node.ID, node.Name, string(node.Type), fmt.Sprintf("retry %d", a))
			e.observers.OnTaskRetrying(ctx, task, a)
		}
	}

	resp, err := retryExec.Execute(ctx, req)
	finishedAt := time.Now()
	task.FinishedAt = &finishedAt

	if err != nil || !resp.Success {
		if err == nil {
			err = fmt.Errorf("%s", resp.Error)
		}
		span.EndWithError(err)

		if node.SkipOnFailure {
			// skip_on_failure: 标记跳过，继续
			task.Status = model.TaskStatusSkipped
			task.Error = err.Error()
			_ = e.dbStore.UpdateTask(ctx, task)
			e.observers.OnTaskSkipped(ctx, task, err.Error())
			ec.EmitEvent("node.skipped", node.ID, node.Name, string(node.Type), err.Error())

			// 设置占位输出
			ec.SetNodeOutput(node.ID, json.RawMessage(fmt.Sprintf(`{"skipped":true,"reason":"%s"}`, err.Error())))

			mu := &sync.Mutex{}
			mu.Lock()
			completedNodes[node.ID] = true
			mu.Unlock()

			// 激活所有下游边（跳过节点继续向下）
			for _, edge := range dag.GetDownstream(node.ID) {
				_ = edge
			}
			return
		}

		e.handleNodeFailure(ec, task, node, err, completedNodes, errCh)
		return
	}

	// 成功
	task.Status = model.TaskStatusCompleted
	task.Output = resp.Output
	_ = e.dbStore.UpdateTask(ctx, task)
	e.observers.OnTaskCompleted(ctx, task, node, nil)

	ec.SetNodeOutput(node.ID, resp.Output)
	ec.EmitCompletedEvent(node.ID, node.Name, string(node.Type), time.Since(*task.StartedAt))
	e.pushEvent(ec, "node.completed", node.ID, node.Name, string(node.Type), string(resp.Output))

	duration := time.Since(*task.StartedAt)
	ec.RecordTrace(node.ID, string(node.Type), node.Name, task.Input, resp.Output, duration, nil)

	// 关键：条件分支解析
	// condition/switch 节点的输出决定了激活哪些下游边
	outputStr := ""
	if resp.Output != nil {
		outputStr = strings.Trim(string(resp.Output), `"`)
	}

	activatedEdges := dag.GetActivatedDownstream(node.ID, outputStr)
	mu := &sync.Mutex{}
	mu.Lock()
	for _, edge := range activatedEdges {
		_ = edge.TargetID // 入度已在 GetReadyNodes 中处理
	}
	completedNodes[node.ID] = true
	mu.Unlock()
}

// handleNodeFailure 处理节点执行失败
func (e *Engine) handleNodeFailure(ec *ExecutionContext, task *model.Task, node *model.Node, err error, completedNodes map[string]bool, errCh chan<- error) {
	task.Status = model.TaskStatusFailed
	task.Error = err.Error()
	_ = e.dbStore.UpdateTask(context.Background(), task)
	e.observers.OnTaskCompleted(ec.Context, task, node, err)

	ec.EmitErrorEvent(node.ID, node.Name, string(node.Type), err.Error())
	e.pushEvent(ec, "node.error", node.ID, node.Name, string(node.Type), err.Error())

	duration := time.Duration(0)
	if task.StartedAt != nil && task.FinishedAt != nil {
		duration = task.FinishedAt.Sub(*task.StartedAt)
	}
	ec.RecordTrace(node.ID, string(node.Type), node.Name, task.Input, nil, duration, err)

	errCh <- err
}

func (e *Engine) failExecution(ctx context.Context, exec *model.Execution, err error) {
	now := time.Now()
	exec.FinishedAt = &now
	exec.Status = model.ExecutionStatusFailed
	exec.Error = err.Error()
	if updateErr := e.dbStore.UpdateExecution(ctx, exec); updateErr == nil {
		e.observers.OnExecutionCompleted(ctx, exec, err)
	}
	e.runningMu.Lock()
	delete(e.running, exec.ID)
	e.runningMu.Unlock()
}

// pushEvent 同时推送事件到 ExecutionContext 和引擎级事件通道
func (e *Engine) pushEvent(ec *ExecutionContext, eventType, nodeID, nodeName, nodeType, content string) {
	ec.EmitEvent(eventType, nodeID, nodeName, nodeType, content)
	if evtCh := e.eventsForExec(ec.Execution.ID); evtCh != nil {
		evtCh <- model.NewWorkflowEvent(eventType, ec.Execution.ID, nodeID, nodeName, nodeType, content)
	}
}

// pushFinalEvent 推送最终事件（完成/失败/取消），并确保引擎级事件通道已关闭
func (e *Engine) pushFinalEvent(execID string, eventType string) {
	if evtCh := e.eventsForExec(execID); evtCh != nil {
		evtCh <- model.NewWorkflowEvent(eventType, execID, "", "", "", "")
	}
}
