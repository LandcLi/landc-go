package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/workflow/pkg/model"
)

func BenchmarkMemoryStoreExecutionWriteRead(b *testing.B) {
	ms := NewMemoryStore()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exec := &model.Execution{
			ID:         fmt.Sprintf("exec-%d", i),
			WorkflowID: "wf-1",
			Status:     model.ExecutionStatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		_ = ms.CreateExecution(ctx, exec)
		_, _ = ms.GetExecution(ctx, exec.ID)
	}
}

func BenchmarkMemoryStoreTaskWrite(b *testing.B) {
	ms := NewMemoryStore()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task := &model.Task{
			ID:          fmt.Sprintf("task-%d", i),
			ExecutionID: "exec-1",
			NodeID:      "n1",
			Status:      model.TaskStatusPending,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		_ = ms.CreateTask(ctx, task)
	}
}
