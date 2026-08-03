package model

import (
	"testing"
	"time"

	sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestTableNames 验证所有 SaaS 表名映射
func TestTableNames(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"Tenant", (&Tenant{}).TableName(), "saas_tenants"},
		{"DataAccess", (&DataAccess{}).TableName(), "saas_data_access"},
		{"DataOwnership", (&DataOwnership{}).TableName(), "saas_data_ownership"},
		{"DataShareLog", (&DataShareLog{}).TableName(), "saas_data_share_log"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s TableName() = %q, want %q", c.name, c.got, c.want)
		}
	}
}

// TestTenantIsRoot 验证根租户判定
func TestTenantIsRoot(t *testing.T) {
	if root := (&Tenant{}); !root.IsRoot() {
		t.Error("tenant with nil ParentID should be root")
	}
	parent := uint64(1)
	if sub := (&Tenant{ParentID: &parent}); sub.IsRoot() {
		t.Error("tenant with ParentID should not be root")
	}
}

// TestTenantHasChildren 验证子租户判定
func TestTenantHasChildren(t *testing.T) {
	db := openModelTestDB(t)
	db.AutoMigrate(&Tenant{})

	root := &Tenant{ID: 1, Name: "root"}
	if err := db.Create(root).Error; err != nil {
		t.Fatalf("create root: %v", err)
	}
	if root.HasChildren(db) {
		t.Error("root with no children should return false")
	}

	pid := uint64(1)
	child := &Tenant{ID: 2, Name: "child", ParentID: &pid}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}
	if !root.HasChildren(db) {
		t.Error("root with a child should return true")
	}
}

// TestDataAccessIsExpired 验证过期判定
func TestDataAccessIsExpired(t *testing.T) {
	t.Run("nil expire never expired", func(t *testing.T) {
		d := &DataAccess{ExpireAt: nil}
		if d.IsExpired() {
			t.Error("nil ExpireAt should not be expired")
		}
	})

	t.Run("past expire is expired", func(t *testing.T) {
		past := time.Now().Add(-time.Hour)
		d := &DataAccess{ExpireAt: &past}
		if !d.IsExpired() {
			t.Error("past ExpireAt should be expired")
		}
	})

	t.Run("future expire not expired", func(t *testing.T) {
		future := time.Now().Add(time.Hour)
		d := &DataAccess{ExpireAt: &future}
		if d.IsExpired() {
			t.Error("future ExpireAt should not be expired")
		}
	})
}

// TestValueInVariants 验证 valueIn 对不同切片类型的支持
func TestValueInVariants(t *testing.T) {
	if !valueIn("b", []interface{}{"a", "b"}) {
		t.Error("[]interface{} match failed")
	}
	if valueIn("z", []interface{}{"a", "b"}) {
		t.Error("[]interface{} miss should return false")
	}
	if !valueIn("b", []string{"a", "b"}) {
		t.Error("[]string match failed")
	}
	if !valueIn(2, []int{1, 2, 3}) {
		t.Error("[]int match failed")
	}
	if !valueIn(2.5, []float64{1.5, 2.5}) {
		t.Error("[]float64 match failed")
	}
	// 未知类型
	if valueIn("x", "not-a-list") {
		t.Error("unknown list type should return false")
	}
}

// TestCompareNumericEdge 验证数值比较的边界
func TestCompareNumericEdge(t *testing.T) {
	// 非数值 actual → -2
	if got := compareNumeric("abc", 1); got != -2 {
		t.Errorf("non-numeric actual = %d, want -2", got)
	}
	// 非数值 expected → 2
	if got := compareNumeric(1, "abc"); got != 2 {
		t.Errorf("non-numeric expected = %d, want 2", got)
	}
	// 数值相等
	if got := compareNumeric(int64(5), float64(5)); got != 0 {
		t.Errorf("equal values = %d, want 0", got)
	}
	// 不同类型数值
	if got := compareNumeric(uint64(10), float32(9)); got != 1 {
		t.Errorf("uint64 > float32 = %d, want 1", got)
	}
}

// TestToFloat64Variants 验证数值类型转换覆盖
func TestToFloat64Variants(t *testing.T) {
	variants := []interface{}{
		int(1), int32(2), int64(3), uint(4), uint64(5), float32(6), float64(7),
	}
	for i, v := range variants {
		got, ok := toFloat64(v)
		if !ok || got != float64(i+1) {
			t.Errorf("toFloat64(%T) = %v, %v; want %d, true", v, got, ok, i+1)
		}
	}
	if _, ok := toFloat64("not-number"); ok {
		t.Error("toFloat64(string) should fail")
	}
}

// openModelTestDB 打开内存 sqlite 供 model 测试使用
func openModelTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(
		sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&loc=auto"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}
