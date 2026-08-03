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
	if idempCheck == nil {
		idempCheck = idempotent.NewMemoryIdempotencyChecker(24 * time.Hour)
	}
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

// SubscribeEvents 订阅执行事件（别名，兼容旧调用方）
func (e *Engine) SubscribeEvents(ctx context.Context, execID string) (<-chan *model.WorkflowEvent, error) {
	return e.Events(execID)
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

// RunNode 单节点运行（REQ-001）。
// 不经过 DAG 引擎，直接调用节点执行器，用于前端调试。
func (e *Engine) RunNode(ctx context.Context, workflowID, nodeID string, inputs map[string]string) (*executor.ExecuteResponse, error) {
	wf, err := e.dbStore.GetWorkflowWithNodes(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("load workflow: %w", err)
	}

	var targetNode *model.Node
	for _, n := range wf.Nodes {
		if n.ID == nodeID {
			targetNode = n
			break
		}
	}
	if targetNode == nil {
		return nil, fmt.Errorf("node %s not found in workflow %s", nodeID, workflowID)
	}

	execImpl := e.execReg.GetByTypeStr(string(targetNode.Type))
	if execImpl == nil {
		return nil, fmt.Errorf("no executor for node type: %s", targetNode.Type)
	}

	// 构建模拟输入
	input := e.buildMockInput(inputs)

	req := &executor.ExecuteRequest{
		NodeID:   nodeID,
		NodeName: targetNode.Name,
		NodeType: string(targetNode.Type),
		Config:   targetNode.Config,
		Input:    input,
	}
	resp, err := execImpl.Execute(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// buildMockInput 从 map 构建模拟输入（单节点调试用）
func (e *Engine) buildMockInput(inputs map[string]string) json.RawMessage {
	if len(inputs) == 0 {
		return json.RawMessage(`{}`)
	}
	m := make(map[string]string)
	for k, v := range inputs {
		m[k] = v
	}
	data, _ := json.Marshal(m)
	return data
}

// GetExecutionTracing 获取执行的追踪树（REQ-002）。
// 返回树形结构，按父节点/分支/迭代组织，供前端树形面板展示。
func (e *Engine) GetExecutionTracing(ctx context.Context, execID string) ([]*model.TracingNode, error) {
	tasks, err := e.dbStore.ListTasks(ctx, execID)
	if err != nil {
		return nil, err
	}

	// 构建 nodeId → Tasks 的查找表
	type taskInfo struct {
		Status    string
		ElapsedMs int64
		Output    interface{}
		Error     string
	}
	nodeTasks := make(map[string]*taskInfo)
	for _, t := range tasks {
		var elapsed int64
		if t.StartedAt != nil && t.FinishedAt != nil {
			elapsed = t.FinishedAt.Sub(*t.StartedAt).Milliseconds()
		}
		var output interface{}
		if t.Output != nil {
			_ = json.Unmarshal(t.Output, &output)
		}
		nodeTasks[t.NodeID] = &taskInfo{
			Status:    string(t.Status),
			ElapsedMs: elapsed,
			Output:    output,
			Error:     t.Error,
		}
	}

	// 按 parent 组织：key = parentNodeID, branchID 或 ""(根)
	children := make(map[tracingChildKey][]*model.TracingNode)

	// 获取工作流定义获取节点列表
	exec, err := e.dbStore.GetExecution(ctx, execID)
	if err != nil {
		return nil, err
	}
	wf, err := e.dbStore.GetWorkflowWithNodes(ctx, exec.WorkflowID)
	if err != nil {
		return nil, err
	}

	for _, n := range wf.Nodes {
		info := nodeTasks[n.ID]
		if info == nil {
			continue
		}
		tn := &model.TracingNode{
			ID:        n.ID,
			Name:      n.Name,
			Type:      string(n.Type),
			Status:    info.Status,
			ElapsedMs: info.ElapsedMs,
			Output:    info.Output,
			Error:     info.Error,
		}

		// 查找上游（父节点）
		parentID := ""
		branchID := ""
		for _, edge := range wf.Edges {
			if edge.TargetID == n.ID {
				parentNode := findNodeByID(wf, edge.SourceID)
				if parentNode != nil && (parentNode.Type == model.NodeTypeCondition || parentNode.Type == model.NodeTypeSwitch) {
					parentID = edge.SourceID
					branchID = edge.SourcePort
				}
			}
		}

		ck := tracingChildKey{parentID: parentID, branchID: branchID}
		children[ck] = append(children[ck], tn)
	}

	// 组装树：根节点是 parentID == "" 的
	var roots []*model.TracingNode
	for _, tn := range children[tracingChildKey{}] {
		attachChildren(tn, children)
		roots = append(roots, tn)
	}

	return roots, nil
}

type tracingChildKey struct {
	parentID string
	branchID string
}

func attachChildren(parent *model.TracingNode, children map[tracingChildKey][]*model.TracingNode) {
	// 查找该节点作为父节点的直接子节点
	ck := tracingChildKey{parentID: parent.ID}
	for _, child := range children[ck] {
		attachChildren(child, children)
		parent.Children = append(parent.Children, child)
	}
	// 带分支的子节点
	for ck, nodes := range children {
		if ck.parentID == parent.ID && ck.branchID != "" {
			for _, child := range nodes {
				attachChildren(child, children)
				child.BranchID = ck.branchID
				parent.Children = append(parent.Children, child)
			}
		}
	}
}

func findNodeByID(wf *model.Workflow, nodeID string) *model.Node {
	for _, n := range wf.Nodes {
		if n.ID == nodeID {
			return n
		}
	}
	return nil
}

// WaitExecution 阻塞直到执行完成、失败或上下文取消。
// 使用纯轮询方式，不与 Events() 消费竞争。
func (e *Engine) WaitExecution(ctx context.Context, execID string) (*model.Execution, error) {
	exec, err := e.dbStore.GetExecution(ctx, execID)
	if err != nil {
		return nil, err
	}
	if exec.IsFinal() {
		return exec, nil
	}

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

//nolint:gocyclo // DAG 执行主循环（含暂停/恢复/并发批处理），拆分收益低
func (e *Engine) executeDAG(ec *ExecutionContext, dag *DAGGraph, completedNodes map[string]bool, taskMap map[string]*model.Task) error {
	ctx := ec.Context
	execID := ec.Execution.ID
	sem := make(chan struct{}, e.config.MaxParallelTasks)
	var wg sync.WaitGroup
	errCh := make(chan error, len(dag.workflow.Nodes))

	// completedNodes 与 taskMap 被多个 executeNode goroutine 并发访问，必须使用共享锁保护
	completedMu := &sync.Mutex{}
	taskMu := &sync.Mutex{}

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
			e.executeNode(ec, dag, n, completedNodes, completedMu, taskMap, taskMu, errCh)
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
					e.logger.WithContext(context.Background()).Warn("[engine] pause timeout, auto-canceling",
						facade.Field{Key: "execution_id", Value: execID},
						facade.Field{Key: "pause_timeout", Value: e.config.PauseTimeout.String()},
					)
					_ = e.CancelWorkflow(context.Background(), execID)
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
				e.executeNode(ec, dag, n, completedNodes, completedMu, taskMap, taskMu, errCh)
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
func (e *Engine) executeNode(ec *ExecutionContext, dag *DAGGraph, node *model.Node, completedNodes map[string]bool, completedMu *sync.Mutex, taskMap map[string]*model.Task, taskMu *sync.Mutex, errCh chan<- error) { //nolint:gocyclo // 节点执行状态机分支多，拆分会破坏可读性
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

	// 获取或创建任务（taskMap 并发访问需加锁）
	taskMu.Lock()
	task, ok := taskMap[node.ID]
	if ok && task.IsFinal() {
		completedMu.Lock()
		completedNodes[node.ID] = true
		completedMu.Unlock()
		taskMu.Unlock()
		return
	}
	taskMu.Unlock()

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
		taskMu.Lock()
		taskMap[node.ID] = task
		taskMu.Unlock()
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
		e.handleNodeFailure(ec, task, node, err, completedNodes, completedMu, errCh)
		return
	}

	// 使用重试装饰器
	retryStrategy := executor.NewRetryStrategy(node.RetryMode, node.MaxRetries, node.RetryDelay, node.RetryMaxDelay)
	retryExec := executor.NewRetryableExecutor(execImpl, retryStrategy, true)

	// 构建节点引用（供执行器感知图拓扑）
	var nodeRefs []executor.NodeRef
	var edgeRefs []executor.EdgeRef
	if ec.Workflow != nil {
		for _, n := range ec.Workflow.Nodes {
			nodeRefs = append(nodeRefs, executor.NodeRef{ID: n.ID, Name: n.Name, Type: string(n.Type)})
		}
		for _, e := range ec.Workflow.Edges {
			if e.Internal {
				continue
			}
			edgeRefs = append(edgeRefs, executor.EdgeRef{SourceID: e.SourceID, TargetID: e.TargetID})
		}
	}

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
		TriggerID:   ec.Execution.TriggerID,
		WorkflowID:  ec.Execution.WorkflowID,
		NodeRefs:    nodeRefs,
		EdgeRefs:    edgeRefs,
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
			ec.SetNodeOutput(node.ID, json.RawMessage(fmt.Sprintf(`{"skipped":true,"reason":%q}`, err.Error())))

			completedMu.Lock()
			completedNodes[node.ID] = true
			completedMu.Unlock()

			// 激活所有下游边（跳过节点继续向下）
			for _, edge := range dag.GetDownstream(node.ID) {
				_ = edge
			}
			return
		}

		e.handleNodeFailure(ec, task, node, err, completedNodes, completedMu, errCh)
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
	completedMu.Lock()
	for _, edge := range activatedEdges {
		_ = edge.TargetID // 入度已在 GetReadyNodes 中处理
	}
	completedNodes[node.ID] = true
	completedMu.Unlock()
}

// handleNodeFailure 处理节点执行失败
func (e *Engine) handleNodeFailure(ec *ExecutionContext, task *model.Task, node *model.Node, err error, completedNodes map[string]bool, completedMu *sync.Mutex, errCh chan<- error) {
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

// pushFinalEvent 推送最终事件并关闭事件通道（解决"事件通道永不关闭"问题）
func (e *Engine) pushFinalEvent(execID, eventType string) {
	if evtCh := e.eventsForExec(execID); evtCh != nil {
		select {
		case evtCh <- model.NewWorkflowEvent(eventType, execID, "", "", "", ""):
		default:
		}
		close(evtCh)
	}
}
