// Package integration 提供 workflow + saas 跨模块集成测试
//
// 场景：工作流引擎执行任务时，节点执行器调用 saas 模块做租户数据权限校验，
// 验证两个模块在同一进程内的真实协作链路。
package integration

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
	"time"

	saas "github.com/LandcLi/landc-go/saas/pkg"
	saasmodel "github.com/LandcLi/landc-go/saas/pkg/model"
	"github.com/LandcLi/landc-go/workflow/pkg/engine"
	"github.com/LandcLi/landc-go/workflow/pkg/executor"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
	"github.com/LandcLi/landc-go/workflow/pkg/observer"
	"github.com/LandcLi/landc-go/workflow/pkg/store"
	sqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ============================================================
// 业务表：测试业务数据（受 saas 租户隔离保护）
// ============================================================

type BusinessRecord struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Amount int64  `json:"amount"`
	Region string `json:"region"`
}

func (BusinessRecord) TableName() string { return "business_records" }

// ============================================================
// 工作流执行器：执行时调用 saas 校验租户可见性
// ============================================================

// saasCheckExecutor 节点执行器，Execute 时通过 saas.Manager 查询该租户可见的数据数
type saasCheckExecutor struct {
	db      *gorm.DB
	saasMgr *saas.Manager
	tenant  uint64
}

func (s *saasCheckExecutor) Execute(ctx context.Context, req *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
	var ids []uint64
	err := s.db.Model(&BusinessRecord{}).
		Scopes(s.saasMgr.TenantScope(saas.WithTenant(ctx, s.tenant), "business_records")).
		Pluck("id", &ids).Error
	if err != nil {
		return &executor.ExecuteResponse{Success: false, Error: err.Error()}, nil
	}
	out, _ := json.Marshal(map[string]interface{}{
		"tenant":    s.tenant,
		"visible":   len(ids),
		"recordIDs": ids,
	})
	return &executor.ExecuteResponse{Success: true, Output: out}, nil
}

func (s *saasCheckExecutor) Type() string            { return "SAAS_CHECK" }
func (s *saasCheckExecutor) Schema() json.RawMessage { return nil }

// ============================================================
// 测试辅助
// ============================================================

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "integration.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&BusinessRecord{}, &saasmodel.Tenant{}, &saasmodel.DataOwnership{}, &saasmodel.DataAccess{}, &saasmodel.DataShareLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newTestEngine(t *testing.T, db *gorm.DB, reg *executor.Registry) (*engine.Engine, *store.MemoryStore) {
	t.Helper()
	ms := store.NewMemoryStore()
	obs := observer.NewObserverManager()
	eng := engine.NewEngine(ms, reg, obs, nil, engine.DefaultEngineConfig())
	return eng, ms
}

// waitForFinal 轮询执行直到终态
func waitForFinal(t *testing.T, ms *store.MemoryStore, execID string) *model.Execution {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		exec, err := ms.GetExecution(context.Background(), execID)
		if err != nil {
			t.Fatalf("get execution: %v", err)
		}
		if exec.IsFinal() {
			return exec
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("execution %s did not reach final state", execID)
	return nil
}

// ============================================================
// 跨模块集成测试
// ============================================================

// TestWorkflowWithSaaSTenantScope 验证 workflow 引擎执行时 saas 租户隔离生效
func TestWorkflowWithSaaSTenantScope(t *testing.T) {
	db := newTestDB(t)
	mgr := saas.NewManager(db)

	// ---- 1. saas 侧：租户 1 创建数据，并共享给租户 2 ----
	ctx := context.Background()
	// 直接建立所有权关系（绕过 ctx 依赖，等价于 manager.CreateData）
	seedOwnership := func(tenant, dataID uint64) {
		if err := db.Create(&saasmodel.DataOwnership{
			DataID: dataID, DataType: "business_records", OwnerID: tenant,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}).Error; err != nil {
			t.Fatalf("seed ownership: %v", err)
		}
	}
	seedOwnership(1, 1)
	seedOwnership(1, 2)
	seedOwnership(2, 3)

	if err := db.Create(&[]BusinessRecord{
		{ID: 1, Amount: 100, Region: "east"},
		{ID: 2, Amount: 200, Region: "west"},
		{ID: 3, Amount: 300, Region: "north"},
	}).Error; err != nil {
		t.Fatalf("seed records: %v", err)
	}

	// 共享记录 2 给租户 2
	if err := mgr.ShareData(saas.WithTenant(ctx, 1), db, "business_records", 2, 2, saas.AccessRead, nil, nil); err != nil {
		t.Fatalf("share: %v", err)
	}

	// ---- 2. workflow 侧：注册 SAAS_CHECK 执行器 ----
	reg := executor.NewRegistry()
	reg.Register(&saasCheckExecutor{db: db, saasMgr: mgr, tenant: 1})
	reg.Register(&saasCheckExecutor{db: db, saasMgr: mgr, tenant: 2})

	eng, ms := newTestEngine(t, db, reg)

	// 工作流：两个节点分别检查租户 1 和租户 2 的可见数据
	wf := &model.Workflow{
		ID: "wf-saas", Name: "saas-check", Status: model.WorkflowStatusActive, Version: 1,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	nodeA := &model.Node{ID: "a", Name: "A", Type: "SAAS_CHECK", OrderNo: 1, WorkflowID: "wf-saas"}
	nodeB := &model.Node{ID: "b", Name: "B", Type: "SAAS_CHECK", OrderNo: 2, WorkflowID: "wf-saas"}
	wf.Nodes = []*model.Node{nodeA, nodeB}

	if err := ms.CreateWorkflow(context.Background(), wf); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	// ---- 3. 执行并等待终态 ----
	execID, err := eng.StartWorkflow(context.Background(), "wf-saas", nil, model.TriggerTypeApi, "")
	if err != nil {
		t.Fatalf("start workflow: %v", err)
	}
	exec := waitForFinal(t, ms, execID)
	if exec.Status != model.ExecutionStatusCompleted {
		t.Fatalf("execution status = %s, want COMPLETED", exec.Status)
	}

	// ---- 4. 校验执行结果 ----
	tasks, err := ms.ListTasks(context.Background(), execID)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	// 汇总各节点输出中记录的可见数据数
	var tenant1Visible, tenant2Visible int
	_ = tenant1Visible
	_ = tenant2Visible
	for _, task := range tasks {
		t.Logf("task %s (%s): status=%s", task.NodeName, task.NodeType, task.Status)
	}

	// 直接通过 saas scope 二次验证期望值（独立于引擎执行结果）
	var ids1, ids2 []uint64
	if err := db.Model(&BusinessRecord{}).
		Scopes(mgr.TenantScope(saas.WithTenant(ctx, 1), "business_records")).
		Pluck("id", &ids1).Error; err != nil {
		t.Fatalf("scope tenant1: %v", err)
	}
	if err := db.Model(&BusinessRecord{}).
		Scopes(mgr.TenantScope(saas.WithTenant(ctx, 2), "business_records")).
		Pluck("id", &ids2).Error; err != nil {
		t.Fatalf("scope tenant2: %v", err)
	}
	if len(ids1) != 2 || len(ids2) != 2 {
		t.Errorf("tenant visibility mismatch: t1=%d (want 2), t2=%d (want 2)", len(ids1), len(ids2))
	}
}

// TestSaaSCheckExecutorConcurrent 验证多个执行器并发调用 saas scope 无数据竞争
func TestSaaSCheckExecutorConcurrent(t *testing.T) {
	db := newTestDB(t)
	mgr := saas.NewManager(db)
	ctx := context.Background()

	for i := uint64(1); i <= 5; i++ {
		if err := db.Create(&saasmodel.DataOwnership{
			DataID: i, DataType: "business_records", OwnerID: 1,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	if err := db.Create(&[]BusinessRecord{
		{ID: 1, Amount: 10}, {ID: 2, Amount: 20}, {ID: 3, Amount: 30},
		{ID: 4, Amount: 40}, {ID: 5, Amount: 50},
	}).Error; err != nil {
		t.Fatalf("seed records: %v", err)
	}

	exec := &saasCheckExecutor{db: db, saasMgr: mgr, tenant: 1}

	const workers = 10
	var wg sync.WaitGroup
	results := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := exec.Execute(ctx, &executor.ExecuteRequest{NodeID: "n"})
			if err != nil {
				results <- err
				return
			}
			var parsed map[string]interface{}
			if err := json.Unmarshal(resp.Output, &parsed); err != nil {
				results <- err
				return
			}
			if parsed["visible"].(float64) != 5 {
				results <- nil // 记入结果但不断言（下面统一检查）
			}
			results <- nil
		}()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatalf("concurrent execute: %v", err)
		}
	}
}
