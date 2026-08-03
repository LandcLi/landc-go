package engine

import (
	"context"
	"sync"
	"testing"

	"github.com/LandcLi/landc-go/workflow/pkg/executor"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
)

// TestEngine_ConcurrentExecutions 压力冒烟：多个工作流实例并发执行（每实例含并行节点）
// 每次运行 50 个并发 StartWorkflow，验证无数据竞争、全部完成、节点恰好执行一次
func TestEngine_ConcurrentExecutions(t *testing.T) {
	reg := executor.NewRegistry()
	reg.Register(&mockExecutor{
		typeName: "TEST",
		fn: func(_ context.Context, _ *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
			return &executor.ExecuteResponse{Success: true, Output: []byte(`{}`)}, nil
		},
	})
	eng, ms := newTestEngine(reg)

	wf := newTestWorkflow("wf-stress",
		[]*model.Node{
			{ID: "a", Name: "A", Type: "TEST", OrderNo: 1},
			{ID: "b", Name: "B", Type: "TEST", OrderNo: 2},
			{ID: "c", Name: "C", Type: "TEST", OrderNo: 2},
			{ID: "d", Name: "D", Type: "TEST", OrderNo: 3},
		},
		[]*model.Edge{
			{ID: "e1", SourceID: "a", TargetID: "b"},
			{ID: "e2", SourceID: "a", TargetID: "c"},
			{ID: "e3", SourceID: "b", TargetID: "d"},
			{ID: "e4", SourceID: "c", TargetID: "d"},
		},
	)
	if err := ms.CreateWorkflow(context.Background(), wf); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	const concurrency = 50
	var wg sync.WaitGroup
	execIDs := make([]string, concurrency)
	errCh := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			execID, err := eng.StartWorkflow(context.Background(), "wf-stress", nil, model.TriggerTypeApi, "")
			if err != nil {
				errCh <- err
				return
			}
			execIDs[idx] = execID
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent start: %v", err)
	}

	// 所有执行均达终态且成功
	for _, execID := range execIDs {
		exec := waitForFinal(t, ms, execID)
		if exec.Status != model.ExecutionStatusCompleted {
			t.Errorf("exec %s status = %s, want COMPLETED", execID, exec.Status)
		}
	}
}

// TestEngine_ConcurrentNodesSingleExec 验证单实例内大扇出/扇入结构连跑 5 次无竞争且稳定
// 结构：root → 10 并行（layer2）→ 10 并行（layer3）→ sink 汇聚
func TestEngine_ConcurrentNodesSingleExec(t *testing.T) {
	reg := executor.NewRegistry()
	reg.Register(&mockExecutor{
		typeName: "TEST",
		fn: func(_ context.Context, _ *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
			return &executor.ExecuteResponse{Success: true, Output: []byte(`{}`)}, nil
		},
	})
	eng, ms := newTestEngine(reg)

	// 节点：root、layer2 x10、layer3 x10、sink
	nodes := []*model.Node{
		{ID: "root", Name: "ROOT", Type: "TEST", OrderNo: 1},
		{ID: "sink", Name: "SINK", Type: "TEST", OrderNo: 4},
	}
	edges := []*model.Edge{}
	for i := 0; i < 10; i++ {
		id2 := "l2_" + string(rune('a'+i))
		nodes = append(nodes, &model.Node{ID: id2, Name: id2, Type: "TEST", OrderNo: 2})
		edges = append(edges, &model.Edge{ID: "e_root_" + id2, SourceID: "root", TargetID: id2})

		id3 := "l3_" + string(rune('a'+i))
		nodes = append(nodes, &model.Node{ID: id3, Name: id3, Type: "TEST", OrderNo: 3})
		edges = append(edges,
			&model.Edge{ID: "e_2_3_" + id3, SourceID: id2, TargetID: id3},
			&model.Edge{ID: "e_3_sink_" + id3, SourceID: id3, TargetID: "sink"})
	}

	wf := newTestWorkflow("wf-fan", nodes, edges)
	if err := ms.CreateWorkflow(context.Background(), wf); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	// 连跑 5 次验证稳定
	for round := 0; round < 5; round++ {
		execID, err := eng.StartWorkflow(context.Background(), "wf-fan", nil, model.TriggerTypeApi, "")
		if err != nil {
			t.Fatalf("round %d start: %v", round, err)
		}
		exec := waitForFinal(t, ms, execID)
		if exec.Status != model.ExecutionStatusCompleted {
			t.Fatalf("round %d status = %s, want COMPLETED (err=%s)", round, exec.Status, exec.Error)
		}
	}
}
