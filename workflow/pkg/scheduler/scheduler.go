package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/registry"
	"github.com/LandcLi/landc-go/log/facade"
	"github.com/LandcLi/landc-go/workflow/pkg/engine"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
	"github.com/LandcLi/landc-go/workflow/pkg/store"
)

// ============================================================
// Scheduler — 分布式调度器
// 复用：frame/pkg/registry (etcd 服务发现), log/facade (日志)
// ============================================================

type SchedulerConfig struct {
	PollInterval      time.Duration
	BatchSize         int
	HeartbeatInterval time.Duration
	DeadWorkerTimeout time.Duration
	TaskLockTTL       time.Duration
	EnableAutoRemove  bool
	// 服务注册相关
	ServiceName string // 注册到 etcd 的服务名
	ServicePort int    // 当前 Worker 端口
}

func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		PollInterval:      2 * time.Second,
		BatchSize:         20,
		HeartbeatInterval: 10 * time.Second,
		DeadWorkerTimeout: 30 * time.Second,
		TaskLockTTL:       60 * time.Second,
		EnableAutoRemove:  true,
		ServiceName:       "workflow-worker",
		ServicePort:       0,
	}
}

type Scheduler struct {
	dbStore store.Store
	engine  *engine.Engine
	etcdReg registry.Registry // 复用 frame/pkg/registry 的 etcd 注册中心
	logger  facade.Logger

	workerID  string
	workerGrp string
	address   string

	mu      sync.Mutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	config SchedulerConfig
}

func NewScheduler(
	dbStore store.Store,
	eng *engine.Engine,
	etcdReg registry.Registry,
	workerID string,
	group string,
	address string,
	config SchedulerConfig,
) *Scheduler {
	if workerID == "" {
		workerID = fmt.Sprintf("wf-worker-%d", time.Now().UnixNano())
	}
	return &Scheduler{
		dbStore:   dbStore,
		engine:    eng,
		etcdReg:   etcdReg,
		logger:    facade.GetLoggerWithName("workflow.scheduler"),
		workerID:  workerID,
		workerGrp: group,
		address:   address,
		config:    config,
	}
}

// Start 启动调度器 — 通过 etcd 注册 Worker 服务实例
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true

	// 通过 etcd 注册 Worker（可选，nil 时回退到 DB 注册）
	if s.etcdReg != nil {
		instance := &registry.ServiceInstance{
			ID:      s.workerID,
			Name:    s.config.ServiceName,
			Address: s.address,
			Port:    s.config.ServicePort,
			Metadata: map[string]string{
				"group": s.workerGrp,
				"type":  "workflow-worker",
			},
			Weight: 1,
		}
		if err := s.etcdReg.Register(s.ctx, instance); err != nil {
			s.running = false
			return fmt.Errorf("etcd register worker failed: %w", err)
		}
		s.logger.WithContext(s.ctx).Info("[scheduler] worker registered via etcd",
			facade.Field{Key: "worker_id", Value: s.workerID},
			facade.Field{Key: "service", Value: s.config.ServiceName},
			facade.Field{Key: "address", Value: instance.Endpoint()},
		)
	} else {
		// 回退：DB 注册
		worker := &model.Worker{
			ID:        s.workerID,
			Address:   s.address,
			Group:     s.workerGrp,
			Status:    "ACTIVE",
			Heartbeat: time.Now(),
			StartedAt: time.Now(),
		}
		if err := s.dbStore.RegisterWorker(s.ctx, worker); err != nil {
			s.running = false
			return fmt.Errorf("db register worker failed: %w", err)
		}
		s.wg.Add(1)
		go s.dbHeartbeatLoop()
	}

	s.wg.Add(1)
	go s.pollLoop()

	if s.config.EnableAutoRemove {
		s.wg.Add(1)
		go s.cleanupLoop()
	}

	return nil
}

// Stop 停止调度器 — 通过 etcd 注销
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.cancel()
	s.wg.Wait()

	// 通过 etcd 注销（如果之前注册了）
	if s.etcdReg != nil {
		instance := &registry.ServiceInstance{
			ID:      s.workerID,
			Name:    s.config.ServiceName,
			Address: s.address,
		}
		if err := s.etcdReg.Deregister(context.Background(), instance); err != nil {
			s.logger.Error("[scheduler] etcd deregister failed",
				facade.Field{Key: "error", Value: err.Error()},
			)
		}
	}

	s.running = false
	s.logger.Info("[scheduler] stopped",
		facade.Field{Key: "worker_id", Value: s.workerID},
	)
}

func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// ============================================================
// 内部循环
// ============================================================

func (s *Scheduler) dbHeartbeatLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.config.HeartbeatInterval)
	defer ticker.Stop()

	_ = s.dbStore.UpdateWorkerHeartbeat(s.ctx, s.workerID)

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			_ = s.dbStore.UpdateWorkerHeartbeat(s.ctx, s.workerID)
		}
	}
}

func (s *Scheduler) pollLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.pollAndDispatch()
		}
	}
}

func (s *Scheduler) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(2 * s.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			_ = s.dbStore.RemoveDeadWorkers(s.ctx, int64(s.config.DeadWorkerTimeout.Seconds()))
		}
	}
}

func (s *Scheduler) pollAndDispatch() {
	tasks, err := s.dbStore.GetPendingTasks(s.ctx, time.Now().Unix(), s.config.BatchSize)
	if err != nil {
		s.logger.Error("[scheduler] poll tasks failed",
			facade.Field{Key: "error", Value: err.Error()},
		)
		return
	}

	for _, task := range tasks {
		go s.dispatchTask(task)
	}
}

func (s *Scheduler) dispatchTask(task *model.Task) {
	exec, err := s.dbStore.GetExecution(s.ctx, task.ExecutionID)
	if err != nil {
		return
	}

	if exec.IsFinal() || exec.Status == model.ExecutionStatusPaused {
		task.Status = model.TaskStatusSkipped
		_ = s.dbStore.UpdateTask(s.ctx, task)
		return
	}

	// 在分布式场景下，这里通过 etcd Resolver 获取其他 Worker 地址，
	// 然后通过 gRPC 调用远程执行。
	// 当前为本地执行，通过 engine 的 DAG 驱动
	_ = exec
}
