package cron

import (
	"context"
	"fmt"
	"sync"

	"github.com/LandcLi/landc-go/mvc/pkg/trace"
	"github.com/LandcLi/landc-go/log/facade"
	"github.com/robfig/cron/v3"
)

// Task 定时任务
type Task struct {
	Name     string
	Spec     string // cron 表达式
	Func     func(ctx context.Context) error
	entryID  cron.EntryID
}

// Scheduler 定时任务调度器
type Scheduler struct {
	cron  *cron.Cron
	tasks map[string]*Task
	mu    sync.RWMutex
}

var (
	globalScheduler *Scheduler
	schedulerMu     sync.RWMutex
)

// NewScheduler 创建调度器
func NewScheduler() *Scheduler {
	return &Scheduler{
		cron:  cron.New(cron.WithSeconds()),
		tasks: make(map[string]*Task),
	}
}

// InitGlobalScheduler 初始化全局调度器
func InitGlobalScheduler() *Scheduler {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()

	if globalScheduler == nil {
		globalScheduler = NewScheduler()
	}
	return globalScheduler
}

// GetScheduler 获取全局调度器
func GetScheduler() *Scheduler {
	schedulerMu.RLock()
	defer schedulerMu.RUnlock()
	return globalScheduler
}

// AddTask 添加定时任务
func (s *Scheduler) AddTask(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.Name]; exists {
		return fmt.Errorf("task '%s' already exists", task.Name)
	}

	entryID, err := s.cron.AddFunc(task.Spec, func() {
		s.executeTask(task)
	})
	if err != nil {
		return fmt.Errorf("add cron task '%s' failed: %w", task.Name, err)
	}

	task.entryID = entryID
	s.tasks[task.Name] = task

	facade.GetLogger().WithFields(
		facade.Field{Key: "task", Value: task.Name},
		facade.Field{Key: "spec", Value: task.Spec},
	).Info("cron task registered")

	return nil
}

// RemoveTask 移除定时任务
func (s *Scheduler) RemoveTask(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[name]
	if !exists {
		return fmt.Errorf("task '%s' not found", name)
	}

	s.cron.Remove(task.entryID)
	delete(s.tasks, name)
	return nil
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.cron.Start()
	facade.Info("cron scheduler started")
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	facade.Info("cron scheduler stopped")
}

// ListTasks 列出所有任务
func (s *Scheduler) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

func (s *Scheduler) executeTask(task *Task) {
	// 每次任务执行创建独立的 trace context
	ctx := trace.InitTrace(context.Background())
	traceID := trace.TraceID(ctx)

	facade.GetLogger().WithFields(
		facade.Field{Key: "task", Value: task.Name},
		facade.Field{Key: "trace_id", Value: traceID},
	).Debug("cron task executing")

	if err := task.Func(ctx); err != nil {
		facade.GetLogger().WithFields(
			facade.Field{Key: "task", Value: task.Name},
			facade.Field{Key: "trace_id", Value: traceID},
			facade.Field{Key: "error", Value: err.Error()},
		).Error("cron task failed")
	} else {
		facade.GetLogger().WithFields(
			facade.Field{Key: "task", Value: task.Name},
			facade.Field{Key: "trace_id", Value: traceID},
		).Debug("cron task completed")
	}
}

// ============ 便捷函数 ============

// AddFunc 便捷添加函数任务
func AddFunc(name, spec string, fn func(ctx context.Context) error) error {
	scheduler := GetScheduler()
	if scheduler == nil {
		scheduler = InitGlobalScheduler()
	}
	return scheduler.AddTask(&Task{
		Name: name,
		Spec: spec,
		Func: fn,
	})
}

// Start 启动全局调度器
func Start() {
	if s := GetScheduler(); s != nil {
		s.Start()
	}
}

// Stop 停止全局调度器
func Stop() {
	if s := GetScheduler(); s != nil {
		s.Stop()
	}
}
