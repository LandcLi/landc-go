package saas

import (
	"context"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/saas/pkg/model"
	"gorm.io/gorm"
)

// TestCreateDataMissingTenant 验证无租户上下文时创建失败
func TestCreateDataMissingTenant(t *testing.T) {
	_, m := setupTestDB(t)
	tx := setupDBForTest(t, m)
	defer tx.Rollback()

	if err := m.CreateData(context.Background(), tx, "test_business_data", 99); err == nil {
		t.Fatal("CreateData without tenant context should fail")
	}
}

// TestShareDataAuthorization 验证非 owner 共享失败 + 带过期时间的共享
func TestShareDataAuthorization(t *testing.T) {
	db, m := setupTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()

	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 1); err != nil {
		t.Fatalf("create: %v", err)
	}

	// 非 owner（租户 3）尝试共享 → 失败
	if err := m.ShareData(tenantCtx(3), tx, "test_business_data", 1, 2, AccessRead, nil, nil); err == nil {
		t.Fatal("non-owner share should fail")
	}

	// owner 带过期时间共享
	future := time.Now().Add(24 * time.Hour)
	if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 1, 2, AccessWrite, &future, nil); err != nil {
		t.Fatalf("share with expiry: %v", err)
	}
	tx.Commit()

	// 验证过期时间已写入
	var access model.DataAccess
	if err := db.Where("data_id = ? AND tenant_id = ?", uint64(1), uint64(2)).First(&access).Error; err != nil {
		t.Fatalf("get access: %v", err)
	}
	if access.ExpireAt == nil {
		t.Error("expire_at should be persisted")
	}
	if access.AccessLevel != AccessWrite {
		t.Errorf("access level = %d, want %d", access.AccessLevel, AccessWrite)
	}

	// 共享已过期的数据给另一租户
	past := time.Now().Add(-time.Hour)
	tx = db.Begin()
	if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 1, 4, AccessRead, &past, nil); err != nil {
		t.Fatalf("share expired: %v", err)
	}
	tx.Commit()

	// GetDataAccessors 不应返回已过期的访问者
	accessors, err := m.GetDataAccessors(context.Background(), "test_business_data", 1)
	if err != nil {
		t.Fatalf("get accessors: %v", err)
	}
	for _, a := range accessors {
		if a.TenantID == 4 {
			t.Error("expired accessor should be filtered out")
		}
	}
}

// TestTransferOwnership 验证所有权转移
func TestTransferOwnership(t *testing.T) {
	db, m := setupTestDB(t)
	tx := db.Begin()
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 1); err != nil {
		t.Fatalf("create: %v", err)
	}
	tx.Commit()

	// 非 owner 转移 → 失败
	tx = db.Begin()
	if err := m.TransferOwnership(tenantCtx(3), tx, "test_business_data", 1, 2); err == nil {
		t.Fatal("non-owner transfer should fail")
	}
	tx.Rollback()

	// owner 转移给租户 2
	tx = db.Begin()
	if err := m.TransferOwnership(tenantCtx(1), tx, "test_business_data", 1, 2); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	tx.Commit()

	owner, err := m.GetDataOwner(context.Background(), "test_business_data", 1)
	if err != nil {
		t.Fatalf("get owner: %v", err)
	}
	if owner != 2 {
		t.Errorf("owner = %d, want 2", owner)
	}

	// 新 owner 可见、原 owner 不可见（原 owner 访问记录已删除）
	var ids []uint64
	if err := db.Model(&TestBusinessData{}).Scopes(m.TenantScope(tenantCtx(2), "test_business_data")).Pluck("id", &ids).Error; err != nil {
		t.Fatalf("scope new owner: %v", err)
	}
	if !contains(ids, 1) {
		t.Errorf("new owner should see data 1, got %v", ids)
	}
	ids = nil
	_ = db.Model(&TestBusinessData{}).Scopes(m.TenantScope(tenantCtx(1), "test_business_data")).Pluck("id", &ids).Error
	if contains(ids, 1) {
		t.Errorf("old owner should not see data 1, got %v", ids)
	}
}

// TestDeleteData 验证数据删除级联清理归属与访问关系
func TestDeleteData(t *testing.T) {
	db, m := setupTestDB(t)
	tx := db.Begin()
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 1); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 1, 2, AccessRead, nil, nil); err != nil {
		t.Fatalf("share: %v", err)
	}
	tx.Commit()

	// 非 owner 删除 → 失败
	tx = db.Begin()
	if err := m.DeleteData(tenantCtx(3), tx, "test_business_data", 1); err == nil {
		t.Fatal("non-owner delete should fail")
	}
	tx.Rollback()

	// owner 删除
	tx = db.Begin()
	if err := m.DeleteData(tenantCtx(1), tx, "test_business_data", 1); err != nil {
		t.Fatalf("delete: %v", err)
	}
	tx.Commit()

	// 归属与访问关系均已清理
	var ownCount, accessCount int64
	db.Model(&model.DataOwnership{}).Where("data_id = ?", uint64(1)).Count(&ownCount)
	db.Model(&model.DataAccess{}).Where("data_id = ?", uint64(1)).Count(&accessCount)
	if ownCount != 0 || accessCount != 0 {
		t.Errorf("ownership/access should be cleaned, own=%d access=%d", ownCount, accessCount)
	}
}

// TestGetDataOwnerNotFound 验证查询不存在数据的 owner 返回错误
func TestGetDataOwnerNotFound(t *testing.T) {
	_, m := setupTestDB(t)
	if _, err := m.GetDataOwner(context.Background(), "test_business_data", 999); err == nil {
		t.Fatal("GetDataOwner for missing data should fail")
	}
}

// TestListTenantDataDefaultPagination 验证默认分页参数
func TestListTenantDataDefaultPagination(t *testing.T) {
	_, m := setupTestDB(t)
	data, total, err := m.ListTenantData(context.Background(), "test_business_data", 999, 0, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 0 || len(data) != 0 {
		t.Errorf("expected empty for unknown tenant, got data=%v total=%d", data, total)
	}

	// pageSize 超限被钳制
	_, _, err = m.ListTenantData(context.Background(), "test_business_data", 999, 1, 5000)
	if err != nil {
		t.Fatalf("list oversized page: %v", err)
	}
}

// TestCleanupExpiredAccess 验证过期访问记录清理
func TestCleanupExpiredAccess(t *testing.T) {
	db, m := setupTestDB(t)
	tx := db.Begin()
	if err := m.CreateData(tenantCtx(1), tx, "test_business_data", 1); err != nil {
		t.Fatalf("create: %v", err)
	}
	past := time.Now().Add(-time.Hour)
	if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 1, 2, AccessRead, &past, nil); err != nil {
		t.Fatalf("share expired: %v", err)
	}
	tx.Commit()

	if err := CleanupExpiredAccess(db); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	var count int64
	db.Model(&model.DataAccess{}).Where("tenant_id = ?", uint64(2)).Count(&count)
	if count != 0 {
		t.Errorf("expired access should be cleaned, count=%d", count)
	}
}

// TestGetTenantDataStats 验证租户数据统计
func TestGetTenantDataStats(t *testing.T) {
	db, m := setupTestDB(t)
	tx := db.Begin()
	for _, id := range []uint64{1, 2} {
		if err := m.CreateData(tenantCtx(1), tx, "test_business_data", id); err != nil {
			t.Fatalf("create %d: %v", id, err)
		}
	}
	if err := m.ShareData(tenantCtx(1), tx, "test_business_data", 1, 2, AccessRead, nil, nil); err != nil {
		t.Fatalf("share: %v", err)
	}
	tx.Commit()

	stats, err := GetTenantDataStats(db, 1)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats["owned"] != 2 {
		t.Errorf("owned = %d, want 2", stats["owned"])
	}
	if stats["accessible"] != 2 {
		t.Errorf("accessible = %d, want 2", stats["accessible"])
	}
	if stats["shared"] != 1 {
		t.Errorf("shared = %d, want 1", stats["shared"])
	}
}

// TestBatchCreateData 验证批量创建数据归属关系
func TestBatchCreateData(t *testing.T) {
	db, m := setupTestDB(t)
	// 业务表补插 10/11/12，否则 scope 无法命中
	if err := db.Create(&[]TestBusinessData{
		{ID: 10, Status: "active", Amount: 100},
		{ID: 11, Status: "active", Amount: 200},
		{ID: 12, Status: "active", Amount: 300},
	}).Error; err != nil {
		t.Fatalf("seed business: %v", err)
	}
	ids := []uint64{10, 11, 12}
	if err := BatchCreateData(db, "test_business_data", 7, ids); err != nil {
		t.Fatalf("batch create: %v", err)
	}

	// 归属记录
	var ownCount int64
	db.Model(&model.DataOwnership{}).Where("owner_id = ?", uint64(7)).Count(&ownCount)
	if ownCount != int64(len(ids)) {
		t.Errorf("ownership count = %d, want %d", ownCount, len(ids))
	}
	// 访问记录
	var accessCount int64
	db.Model(&model.DataAccess{}).Where("tenant_id = ?", uint64(7)).Count(&accessCount)
	if accessCount != int64(len(ids)) {
		t.Errorf("access count = %d, want %d", accessCount, len(ids))
	}

	// 租户 7 通过 scope 可见全部数据
	var scoped []uint64
	if err := db.Model(&TestBusinessData{}).
		Scopes(m.TenantScopeWithOwn(tenantCtx(7), "test_business_data")).
		Pluck("id", &scoped).Error; err != nil {
		t.Fatalf("scope: %v", err)
	}
	for _, id := range ids {
		if !contains(scoped, id) {
			t.Errorf("tenant 7 should own data %d, got %v", id, scoped)
		}
	}
}

// setupDBForTest 开启事务供测试使用
func setupDBForTest(t *testing.T, m *Manager) *gorm.DB {
	t.Helper()
	return m.db.Begin()
}
