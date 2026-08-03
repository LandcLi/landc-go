package cron

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewScheduler(t *testing.T) {
	scheduler := NewScheduler()
	if scheduler == nil {
		t.Fatal("NewScheduler should not return nil")
	}

	if scheduler.cron == nil {
		t.Error("cron field should not be nil")
	}

	if scheduler.tasks == nil {
		t.Error("tasks map should not be nil")
	}
}

func TestInitGlobalScheduler(t *testing.T) {
	scheduler := InitGlobalScheduler()
	if scheduler == nil {
		t.Fatal("InitGlobalScheduler should not return nil")
	}

	scheduler2 := InitGlobalScheduler()
	if scheduler != scheduler2 {
		t.Error("InitGlobalScheduler should return the same instance")
	}
}

func TestGetScheduler(t *testing.T) {
	scheduler := InitGlobalScheduler()

	retrieved := GetScheduler()
	if retrieved != scheduler {
		t.Error("GetScheduler should return the global scheduler")
	}
}

func TestAddTask(t *testing.T) {
	scheduler := NewScheduler()

	task := &Task{
		Name: "test_task",
		Spec: "0 */1 * * * *", // 每分钟
		Func: func(ctx context.Context) error {
			return nil
		},
	}

	err := scheduler.AddTask(task)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	if _, exists := scheduler.tasks["test_task"]; !exists {
		t.Error("Task should be registered")
	}
}

func TestAddTask_Duplicate(t *testing.T) {
	scheduler := NewScheduler()

	task := &Task{
		Name: "duplicate_task",
		Spec: "0 */1 * * * *",
		Func: func(ctx context.Context) error {
			return nil
		},
	}

	err := scheduler.AddTask(task)
	if err != nil {
		t.Fatalf("First AddTask failed: %v", err)
	}

	err = scheduler.AddTask(task)
	if err == nil {
		t.Error("Should not allow duplicate task registration")
	}
}

func TestRemoveTask(t *testing.T) {
	scheduler := NewScheduler()

	task := &Task{
		Name: "remove_task",
		Spec: "0 */1 * * * *",
		Func: func(ctx context.Context) error {
			return nil
		},
	}

	scheduler.AddTask(task)

	err := scheduler.RemoveTask("remove_task")
	if err != nil {
		t.Fatalf("RemoveTask failed: %v", err)
	}

	if _, exists := scheduler.tasks["remove_task"]; exists {
		t.Error("Task should be removed")
	}
}

func TestRemoveTask_NotFound(t *testing.T) {
	scheduler := NewScheduler()

	err := scheduler.RemoveTask("nonexistent_task")
	if err == nil {
		t.Error("Should return error for nonexistent task")
	}
}

func TestStartStop(t *testing.T) {
	scheduler := NewScheduler()

	taskExecuted := false
	task := &Task{
		Name: "start_stop_task",
		Spec: "0 */1 * * * *", // 每分钟
		Func: func(ctx context.Context) error {
			taskExecuted = true
			return nil
		},
	}

	scheduler.AddTask(task)

	scheduler.Start()

	time.Sleep(100 * time.Millisecond)

	scheduler.Stop()

	if taskExecuted {
		t.Error("Task should not be executed in 100ms")
	}
}

func TestListTasks(t *testing.T) {
	scheduler := NewScheduler()

	task1 := &Task{
		Name: "task1",
		Spec: "0 */1 * * * *",
		Func: func(ctx context.Context) error {
			return nil
		},
	}

	task2 := &Task{
		Name: "task2",
		Spec: "0 0 * * * *", // 每小时
		Func: func(ctx context.Context) error {
			return nil
		},
	}

	scheduler.AddTask(task1)
	scheduler.AddTask(task2)

	tasks := scheduler.ListTasks()
	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
}

func TestAddFunc(t *testing.T) {
	scheduler := InitGlobalScheduler()

	err := AddFunc("test_func", "0 */1 * * * *", func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Fatalf("AddFunc failed: %v", err)
	}

	tasks := scheduler.ListTasks()
	found := false
	for _, task := range tasks {
		if task.Name == "test_func" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Task added by AddFunc should be in scheduler")
	}
}

func TestExecuteTask(t *testing.T) {
	scheduler := NewScheduler()

	executed := false
	task := &Task{
		Name: "execute_task",
		Spec: "0 */1 * * * *",
		Func: func(ctx context.Context) error {
			executed = true
			return nil
		},
	}

	scheduler.AddTask(task)

	scheduler.executeTask(task)

	if !executed {
		t.Error("Task function should be executed")
	}
}

func TestExecuteTask_Error(t *testing.T) {
	scheduler := NewScheduler()

	task := &Task{
		Name: "error_task",
		Spec: "0 */1 * * * *",
		Func: func(ctx context.Context) error {
			return context.Canceled
		},
	}

	scheduler.AddTask(task)

	scheduler.executeTask(task)
}

func TestTaskExecution_Concurrent(t *testing.T) {
	scheduler := NewScheduler()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			task := &Task{
				Name: "concurrent_task_" + string('A'+rune(index)),
				Spec: "0 */1 * * * *",
				Func: func(ctx context.Context) error {
					return nil
				},
			}
			scheduler.AddTask(task)
		}(i)
	}

	wg.Wait()

	tasks := scheduler.ListTasks()
	if len(tasks) != 10 {
		t.Errorf("Expected 10 tasks, got %d", len(tasks))
	}
}

func TestTaskWithCronExpression(t *testing.T) {
	scheduler := NewScheduler()

	task := &Task{
		Name: "cron_expr_task",
		Spec: "0 0 12 * * *", // 每天中午12点
		Func: func(ctx context.Context) error {
			return nil
		},
	}

	err := scheduler.AddTask(task)
	if err != nil {
		t.Fatalf("AddTask with cron expression failed: %v", err)
	}

	if scheduler.tasks["cron_expr_task"].Spec != "0 0 12 * * *" {
		t.Error("Cron expression should be preserved")
	}
}

func TestGlobalFunctions(t *testing.T) {
	InitGlobalScheduler()

	err := AddFunc("global_test", "0 */1 * * * *", func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Fatalf("Global AddFunc failed: %v", err)
	}

	Start()

	time.Sleep(100 * time.Millisecond)

	Stop()
}

func TestTask_NilFunc(t *testing.T) {
	scheduler := NewScheduler()

	task := &Task{
		Name: "nil_func_task",
		Spec: "0 */1 * * * *",
		Func: nil,
	}

	err := scheduler.AddTask(task)
	if err == nil {
		t.Error("Should return error for nil function")
	}
}
