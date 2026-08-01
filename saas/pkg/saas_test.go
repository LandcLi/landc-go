package saas

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/saas/pkg/model"
	sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestBusinessData 测试用业务表
type TestBusinessData struct {
	ID     uint64 `gorm:"primaryKey" json:"id"`
	Status string `gorm:"type:varchar(20)" json:"status"`
	Amount int    `json:"amount"`
}

func (TestBusinessData) TableName() string { return "test_business_data" }

// setupTestDB 初始化 sqlite 内存库并迁移表
// 每个测试使用唯一 DSN，避免 shared cache 下索引/表互相冲突
// 返回带启用约束配置的 Manager（用于约束验证测试）
func setupTestDB(t *testing.T) (*gorm.DB, *Manager) {
	t.Helper()
	// loc=auto 保证 time.Time 参数/字段在 sqlite 中正确序列化与比较
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&loc=auto"
	db, err := gorm.Open(
		sqlite.Open(dsn),
		&gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err == nil {
		// 固定单连接，保证内存库一致性
		sqlDB.SetMaxOpenConns(1)
	}

	// 先清理可能残留的表，保证每次迁移干净（内存库可能被多连接共享）
	if err := db.Migrator().DropTable(
		&model.Tenant{}, &model.DataOwnership{}, &model.DataAccess{},
		&model.DataShareLog{}, &TestBusinessData{},
	); err != nil {
		t.Fatalf("drop tables: %v", err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.AutoMigrate(&TestBusinessData{}); err != nil {
		t.Fatalf("auto migrate business table: %v", err)
	}

	// 准备测试数据
	if err := db.Create(&[]TestBusinessData{
		{ID: 1, Status: "active", Amount: 100},
		{ID: 2, Status: "active", Amount: 200},
		{ID: 3, Status: "disabled", Amount: 300},
	}).Error; err != nil {
		t.Fatalf("seed data: %v", err)
	}

	m := NewManager(db, WithConfig(Config{
		EnableHierarchy:  true,
		EnableConstraint: true,
		CleanupInterval:  time.Hour,
	}))
	return db, m
}

func tenantCtx(tenantID uint64) context.Context {
	return WithTenant(context.Background(), tenantID)
}

// TestTenantIsolation 验证租户数据隔离：不同租户只能看到自己的数据
func TestTenantIsolation(t *testing.T) {
	db, m := setupTestDB(t)

	// 租户 1 拥有数据 1、2；租户 2 拥有数据 3
	tx := db.Begin()
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 1); err != nil {
		t.Fatalf("tenant1 create data1: %v", err)
	}
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 2); err != nil {
		t.Fatalf("tenant1 create data2: %v", err)
	}
	if err := m.CreateData(tenantCtx(2), tx, "test_business_data", 3); err != nil {
		t.Fatalf("tenant2 create data3: %v", err)
	}
	tx.Commit()

	var ids1 []uint64
	if err := db.Model(&TestBusinessData{}).Scopes(m.TenantScope(tenantCtx(1), "test_business_data")).Pluck("id", &ids1).Error; err != nil {
		t.Fatalf("scope tenant1: %v", err)
	}
	if len(ids1) != 2 || !contains(ids1, 1) || !contains(ids1, 2) {
		t.Errorf("tenant1 should see data 1,2, got %v", ids1)
	}

	var ids2 []uint64
	if err := db.Model(&TestBusinessData{}).Scopes(m.TenantScope(tenantCtx(2), "test_business_data")).Pluck("id", &ids2).Error; err != nil {
		t.Fatalf("scope tenant2: %v", err)
	}
	if len(ids2) != 1 || !contains(ids2, 3) {
		t.Errorf("tenant2 should see data 3, got %v", ids2)
	}

	// 无租户上下文的 Scope 不应泄露数据
	var ids3 []uint64
	if err := db.Model(&TestBusinessData{}).Scopes(m.TenantScope(context.Background(), "test_business_data")).Pluck("id", &ids3).Error; err != nil {
		t.Fatalf("scope no tenant: %v", err)
	}
	if len(ids3) != 0 {
		t.Errorf("scope without tenant should return nothing, got %v", ids3)
	}
}

// TestSharedDataAccess 验证共享数据可见性与约束过滤
func TestSharedDataAccess(t *testing.T) {
	db, m := setupTestDB(t)

	tx := db.Begin()
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 1); err != nil {
		t.Fatalf("tenant1 create data1: %v", err)
	}
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 2); err != nil {
		t.Fatalf("tenant1 create data2: %v", err)
	}
	// 租户 1 将数据 1 共享给租户 2（无约束）
	if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 1, 2, AccessRead, nil, nil); err != nil {
		t.Fatalf("share data1: %v", err)
	}
	tx.Commit()

	var ids []uint64
	if err := db.Model(&TestBusinessData{}).Scopes(m.TenantScope(tenantCtx(2), "test_business_data")).Pluck("id", &ids).Error; err != nil {
		t.Fatalf("scope tenant2: %v", err)
	}
	if len(ids) != 1 || !contains(ids, 1) {
		t.Errorf("tenant2 should see shared data 1, got %v", ids)
	}
}

// TestConstraintFiltering 验证带约束的共享数据被正确过滤
func TestConstraintFiltering(t *testing.T) {
	db, m := setupTestDB(t)

	// 约束：仅 status=active 且 amount>150 的数据可访问
	constraints := map[string]interface{}{
		"status": "active",
		"amount": map[string]interface{}{"__gt": 150},
	}

	tx := db.Begin()
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 1); err != nil {
		t.Fatalf("tenant1 create data1: %v", err)
	}
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 2); err != nil {
		t.Fatalf("tenant1 create data2: %v", err)
	}
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 3); err != nil {
		t.Fatalf("tenant1 create data3: %v", err)
	}
	// 共享全部 3 条数据，但带约束
	for _, id := range []uint64{1, 2, 3} {
		if err := m.ShareData(tenantCtx(1), tx, "test_business_data", id, 2, AccessRead, nil, constraints); err != nil {
			t.Fatalf("share data %d: %v", id, err)
		}
	}
	tx.Commit()

	// 校验约束确实写入数据库
	var accessRecords []model.DataAccess
	db.Where("tenant_id = ?", uint64(2)).Find(&accessRecords)
	if len(accessRecords) != 3 {
		t.Fatalf("expected 3 access records, got %d", len(accessRecords))
	}
	for _, a := range accessRecords {
		if a.Constraints == "" {
			t.Fatalf("access record %d should have constraints stored", a.DataID)
		}
	}

	var ids []uint64
	if err := db.Model(&TestBusinessData{}).Scopes(m.TenantScopeWithConstraint(tenantCtx(2), "test_business_data")).Pluck("id", &ids).Error; err != nil {
		t.Fatalf("scope tenant2 with constraint: %v", err)
	}

	// 数据2（active, amount=200）满足约束；数据1（amount=100）不满足；数据3（disabled）不满足
	if len(ids) != 1 || !contains(ids, 2) {
		t.Errorf("only data 2 should pass constraint, got %v", ids)
	}
}

// TestRevokeAccess 验证撤销访问后数据不可见
func TestRevokeAccess(t *testing.T) {
	db, m := setupTestDB(t)

	tx := db.Begin()
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 1); err != nil {
		t.Fatalf("tenant1 create data1: %v", err)
	}
	if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 1, 2, AccessRead, nil, nil); err != nil {
		t.Fatalf("share data1: %v", err)
	}
	tx.Commit()

	var ids []uint64
	_ = db.Model(&TestBusinessData{}).Scopes(m.TenantScope(tenantCtx(2), "test_business_data")).Pluck("id", &ids).Error
	if len(ids) != 1 {
		t.Fatalf("tenant2 should see data before revoke, got %v", ids)
	}

	tx = db.Begin()
	if err := m.RevokeAccess(tenantCtx(1), tx, "test_business_data", 1, 2); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	tx.Commit()

	ids = nil
	_ = db.Model(&TestBusinessData{}).Scopes(m.TenantScope(tenantCtx(2), "test_business_data")).Pluck("id", &ids).Error
	if len(ids) != 0 {
		t.Errorf("tenant2 should see nothing after revoke, got %v", ids)
	}
}

// TestConstraintValidationSQLSafety 验证非法约束列名被拒绝（防 SQL 注入）
func TestConstraintValidationSQLSafety(t *testing.T) {
	db, m := setupTestDB(t)

	tx := db.Begin()
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 1); err != nil {
		t.Fatalf("tenant1 create data1: %v", err)
	}
	// 恶意约束列名（SQL 注入尝试）
	malicious := map[string]interface{}{
		"status; DROP TABLE test_business_data; --": "active",
	}
	if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 1, 2, AccessRead, nil, malicious); err != nil {
		t.Fatalf("share with malicious constraint: %v", err)
	}
	tx.Commit()

	var ids []uint64
	if err := db.Model(&TestBusinessData{}).Scopes(m.TenantScopeWithConstraint(tenantCtx(2), "test_business_data")).Pluck("id", &ids).Error; err != nil {
		t.Fatalf("scope with malicious constraint: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("malicious constraint must be rejected, got %v", ids)
	}

	// 表仍然存在
	if !db.Migrator().HasTable(&TestBusinessData{}) {
		t.Error("table should still exist after malicious constraint")
	}
}

func contains(list []uint64, v uint64) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

// TestConcurrentScopeAccess 验证并发 Scope 查询无数据竞争
func TestConcurrentScopeAccess(t *testing.T) {
	db, m := setupTestDB(t)

	tx := db.Begin()
	for i := uint64(1); i <= 3; i++ {
		if err := m.CreateData(tenantCtx(1), tx, "test_business_data", i); err != nil {
			t.Fatalf("create data %d: %v", i, err)
		}
	}
	tx.Commit()

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var ids []uint64
			_ = db.Model(&TestBusinessData{}).Scopes(m.TenantScope(tenantCtx(1), "test_business_data")).Pluck("id", &ids).Error
		}()
	}
	wg.Wait()
}

// TestModelJSON 验证 model 的 JSON 序列化
func TestModelJSON(t *testing.T) {
	now := time.Now()
	access := model.DataAccess{
		DataID:      1,
		DataType:    "test",
		TenantID:    2,
		AccessLevel: AccessRead,
		GrantType:   GrantDirect,
		GrantedBy:   1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	data, err := json.Marshal(access)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(data) == 0 {
		t.Error("marshal result should not be empty")
	}
}
