package saas

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/saas/pkg/model"
	sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestTenantScopeWithOwn 验证仅查询自有数据的 scope
func TestTenantScopeWithOwn(t *testing.T) {
	db, m := setupTestDB(t)

	tx := db.Begin()
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 1); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := m.CreateData(tenantCtx(2), tx, "test_business_data", 2); err != nil {
		t.Fatalf("create: %v", err)
	}
	// 租户 1 共享数据 1 给租户 3（不影响 WithOwn 结果）
	if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 1, 3, AccessRead, nil, nil); err != nil {
		t.Fatalf("share: %v", err)
	}
	tx.Commit()

	var ids []uint64
	if err := db.Model(&TestBusinessData{}).
		Scopes(m.TenantScopeWithOwn(tenantCtx(1), "test_business_data")).
		Pluck("id", &ids).Error; err != nil {
		t.Fatalf("scope: %v", err)
	}
	if len(ids) != 1 || !contains(ids, 1) {
		t.Errorf("WithOwn should return only owned data, got %v", ids)
	}

	// 无租户上下文 → 空
	ids = nil
	_ = db.Model(&TestBusinessData{}).
		Scopes(m.TenantScopeWithOwn(context.Background(), "test_business_data")).
		Pluck("id", &ids).Error
	if len(ids) != 0 {
		t.Errorf("WithOwn without tenant should be empty, got %v", ids)
	}
}

// TestTenantScopeWithAccess 验证仅查询共享数据的 scope（含访问级别过滤）
func TestTenantScopeWithAccess(t *testing.T) {
	db, m := setupTestDB(t)

	tx := db.Begin()
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 1); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 2); err != nil {
		t.Fatalf("create: %v", err)
	}
	// 共享给租户 3：数据 1 只读、数据 2 读写
	if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 1, 3, AccessRead, nil, nil); err != nil {
		t.Fatalf("share 1: %v", err)
	}
	if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 2, 3, AccessWrite, nil, nil); err != nil {
		t.Fatalf("share 2: %v", err)
	}
	tx.Commit()

	// 无级别过滤 → 两条
	var ids []uint64
	if err := db.Model(&TestBusinessData{}).
		Scopes(m.TenantScopeWithAccess(tenantCtx(3), "test_business_data")).
		Pluck("id", &ids).Error; err != nil {
		t.Fatalf("scope: %v", err)
	}
	if len(ids) != 2 || !contains(ids, 1) || !contains(ids, 2) {
		t.Errorf("WithAccess should return both, got %v", ids)
	}

	// 仅写级别 → 只有数据 2
	ids = nil
	if err := db.Model(&TestBusinessData{}).
		Scopes(m.TenantScopeWithAccess(tenantCtx(3), "test_business_data", AccessWrite)).
		Pluck("id", &ids).Error; err != nil {
		t.Fatalf("scope with level: %v", err)
	}
	if len(ids) != 1 || !contains(ids, 2) {
		t.Errorf("WithAccess(write) should return only data 2, got %v", ids)
	}

	// 无租户上下文 → 空
	ids = nil
	_ = db.Model(&TestBusinessData{}).
		Scopes(m.TenantScopeWithAccess(context.Background(), "test_business_data")).
		Pluck("id", &ids).Error
	if len(ids) != 0 {
		t.Errorf("WithAccess without tenant should be empty, got %v", ids)
	}
}

// seedHierarchy 创建层级租户：根租户 1 + 子租户 2 + 孙租户 3
func seedHierarchy(t *testing.T, db *gorm.DB) {
	t.Helper()
	root := &model.Tenant{ID: 1, Name: "root"}
	if err := db.Create(root).Error; err != nil {
		t.Fatalf("create root: %v", err)
	}
	if err := UpdateTenantPath(db, root); err != nil {
		t.Fatalf("update root path: %v", err)
	}
	pid := uint64(1)
	child := &model.Tenant{ID: 2, Name: "child", ParentID: &pid}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}
	if err := UpdateTenantPath(db, child); err != nil {
		t.Fatalf("update child path: %v", err)
	}
	gpid := uint64(2)
	grand := &model.Tenant{ID: 3, Name: "grand", ParentID: &gpid}
	if err := db.Create(grand).Error; err != nil {
		t.Fatalf("create grand: %v", err)
	}
	if err := UpdateTenantPath(db, grand); err != nil {
		t.Fatalf("update grand path: %v", err)
	}
}

// setupDBWithConfig 按自定义配置创建测试库（复用 setupTestDB 的建表/种子逻辑）
func setupDBWithConfig(t *testing.T, cfg Config) *gorm.DB {
	t.Helper()
	dsn := "file:cfg_" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&loc=auto"
	db, err := gorm.Open(
		sqlite.Open(dsn),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.AutoMigrate(&TestBusinessData{}); err != nil {
		t.Fatalf("migrate business: %v", err)
	}
	if err := db.Create(&[]TestBusinessData{
		{ID: 1, Status: "active", Amount: 100},
		{ID: 2, Status: "active", Amount: 200},
		{ID: 3, Status: "disabled", Amount: 300},
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	return db
}

// CreateDataWithTenant 直接以指定租户创建归属记录（跳过 Manager 的 ctx 依赖）
func CreateDataWithTenant(tx *gorm.DB, tenantID, dataID uint64) error {
	return tx.Create(&model.DataOwnership{
		DataID:    dataID,
		DataType:  "test_business_data",
		OwnerID:   tenantID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}).Error
}

// TestTenantScopeWithChildren 验证层级 scope：父租户可见子租户数据
func TestTenantScopeWithChildren(t *testing.T) {
	db, m := setupTestDB(t)
	seedHierarchy(t, db)

	// 租户 2 拥有数据 1；租户 3 拥有数据 2、3
	tx := db.Begin()
	if err := m.CreateData(tenantCtx(2), tx, "test_business_data", 1); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := m.CreateData(tenantCtx(3), tx, "test_business_data", 2); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := m.CreateData(tenantCtx(3), tx, "test_business_data", 3); err != nil {
		t.Fatalf("create: %v", err)
	}
	tx.Commit()

	// 租户 1（根）可见全部（自己无数据 + 子租户 2/3 的数据）
	var ids []uint64
	if err := db.Model(&TestBusinessData{}).
		Scopes(m.TenantScopeWithChildren(tenantCtx(1), "test_business_data")).
		Pluck("id", &ids).Error; err != nil {
		t.Fatalf("scope root: %v", err)
	}
	if len(ids) != 3 || !contains(ids, 1) || !contains(ids, 2) || !contains(ids, 3) {
		t.Errorf("root should see all children data, got %v", ids)
	}

	// 租户 2 可见自己数据 1 + 孙租户 3 的数据 2、3
	ids = nil
	if err := db.Model(&TestBusinessData{}).
		Scopes(m.TenantScopeWithChildren(tenantCtx(2), "test_business_data")).
		Pluck("id", &ids).Error; err != nil {
		t.Fatalf("scope child: %v", err)
	}
	if len(ids) != 3 {
		t.Errorf("tenant2 should see own+descendant data, got %v", ids)
	}
}

// TestTenantScopeWithChildrenHierarchyDisabled 验证层级关闭时仅查自有数据
func TestTenantScopeWithChildrenHierarchyDisabled(t *testing.T) {
	db := setupDBWithConfig(t, Config{EnableHierarchy: false, EnableConstraint: false})
	seedHierarchy(t, db)

	tx := db.Begin()
	if err := CreateDataWithTenant(tx, 2, 1); err != nil {
		t.Fatalf("create ownership: %v", err)
	}
	tx.Commit()

	var ids []uint64
	if err := db.Model(&TestBusinessData{}).
		Scopes(NewManager(db, WithConfig(Config{EnableHierarchy: false})).TenantScopeWithChildren(tenantCtx(1), "test_business_data")).
		Pluck("id", &ids).Error; err != nil {
		t.Fatalf("scope: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("hierarchy disabled: parent should not see child data, got %v", ids)
	}
}

// TestTenantScopeWithConstraintFullBranches 验证约束 scope 的各个分支
func TestTenantScopeWithConstraintFullBranches(t *testing.T) {
	db, m := setupTestDB(t)

	tx := db.Begin()
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 1); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 2); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 3); err != nil {
		t.Fatalf("create: %v", err)
	}
	// 数据 1：无约束共享
	if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 1, 2, AccessRead, nil, nil); err != nil {
		t.Fatalf("share 1: %v", err)
	}
	// 数据 2：非法约束 JSON（保守拒绝）
	invalidConstraint := map[string]interface{}{"amount": map[string]interface{}{"__bogus": 1}}
	if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 2, 2, AccessRead, nil, invalidConstraint); err != nil {
		t.Fatalf("share 2: %v", err)
	}
	// 数据 3：满足约束（amount > 150）
	validConstraint := map[string]interface{}{"amount": map[string]interface{}{"__gt": 150}}
	if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 3, 2, AccessRead, nil, validConstraint); err != nil {
		t.Fatalf("share 3: %v", err)
	}
	tx.Commit()

	var ids []uint64
	if err := db.Model(&TestBusinessData{}).
		Scopes(m.TenantScopeWithConstraint(tenantCtx(2), "test_business_data")).
		Pluck("id", &ids).Error; err != nil {
		t.Fatalf("scope: %v", err)
	}
	// 数据 1（无约束可见）+ 数据 3（满足约束）；数据 2（未知操作符拒绝）
	if len(ids) != 2 || !contains(ids, 1) || !contains(ids, 3) {
		t.Errorf("unexpected visible data: %v", ids)
	}
}

// TestTenantScopeWithConstraintAllOperators 验证约束操作符在 SQL 层的覆盖
func TestTenantScopeWithConstraintAllOperators(t *testing.T) {
	db, m := setupTestDB(t)

	// 给数据 1（amount=100）挂各操作符约束并逐个验证
	operators := []struct {
		name   string
		op     map[string]interface{}
		passes bool
	}{
		{"__eq", map[string]interface{}{"amount": map[string]interface{}{"__eq": 100}}, true},
		{"__ne", map[string]interface{}{"amount": map[string]interface{}{"__ne": 200}}, true},
		{"__gte", map[string]interface{}{"amount": map[string]interface{}{"__gte": 100}}, true},
		{"__lte", map[string]interface{}{"amount": map[string]interface{}{"__lte": 100}}, true},
		{"__lt", map[string]interface{}{"amount": map[string]interface{}{"__lt": 100}}, false},
		{"__in", map[string]interface{}{"amount": map[string]interface{}{"__in": []int{50, 100}}}, true},
		{"__in_miss", map[string]interface{}{"amount": map[string]interface{}{"__in": []int{1, 2}}}, false},
	}

	for _, op := range operators {
		t.Run(op.name, func(t *testing.T) {
			tx := db.Begin()
			if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 1); err != nil {
				t.Fatalf("create: %v", err)
			}
			if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 1, 2, AccessRead, nil, map[string]interface{}{"amount": op.op["amount"]}); err != nil {
				t.Fatalf("share: %v", err)
			}
			tx.Commit()
			defer func() { _ = db.Where("data_id = ?", uint64(1)).Delete(&model.DataAccess{}).Error }()

			var ids []uint64
			if err := db.Model(&TestBusinessData{}).
				Scopes(m.TenantScopeWithConstraint(tenantCtx(2), "test_business_data")).
				Pluck("id", &ids).Error; err != nil {
				t.Fatalf("scope: %v", err)
			}
			visible := contains(ids, 1)
			if visible != op.passes {
				t.Errorf("%s: visible=%v, want %v", op.name, visible, op.passes)
			}
		})
	}
}

// TestTenantScopeWithConstraintNoTenant 验证无租户上下文返回空
func TestTenantScopeWithConstraintNoTenant(t *testing.T) {
	db, m := setupTestDB(t)
	var ids []uint64
	if err := db.Model(&TestBusinessData{}).
		Scopes(m.TenantScopeWithConstraint(context.Background(), "test_business_data")).
		Pluck("id", &ids).Error; err != nil {
		t.Fatalf("scope: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("constraint scope without tenant should be empty, got %v", ids)
	}
}

// TestTenantScopeNoTenant 验证基础 scope 无租户上下文返回空
func TestTenantScopeNoTenant(t *testing.T) {
	db, m := setupTestDB(t)
	var ids []uint64
	if err := db.Model(&TestBusinessData{}).
		Scopes(m.TenantScope(context.Background(), "test_business_data")).
		Pluck("id", &ids).Error; err != nil {
		t.Fatalf("scope: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("scope without tenant should be empty, got %v", ids)
	}
}
