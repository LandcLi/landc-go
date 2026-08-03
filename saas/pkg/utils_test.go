package saas

import (
	"strings"
	"testing"

	"github.com/LandcLi/landc-go/saas/pkg/model"
	sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestParseConstraint(t *testing.T) {
	t.Run("empty string returns nil", func(t *testing.T) {
		c, err := ParseConstraint("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c != nil {
			t.Errorf("expected nil constraint, got %v", c)
		}
	})

	t.Run("valid json", func(t *testing.T) {
		c, err := ParseConstraint(`{"status":"active","amount":100}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c["status"] != "active" {
			t.Errorf("expected status=active, got %v", c["status"])
		}
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		if _, err := ParseConstraint(`{invalid`); err == nil {
			t.Error("expected error for invalid json")
		}
	})
}

func TestValidateConstraint(t *testing.T) {
	data := map[string]interface{}{
		"status": "active",
		"amount": 100,
		"tags":   "a,b,c",
	}

	t.Run("simple equality pass", func(t *testing.T) {
		ok := ValidateConstraint(data, map[string]interface{}{"status": "active"})
		if !ok {
			t.Error("expected constraint to pass")
		}
	})

	t.Run("missing field fails", func(t *testing.T) {
		ok := ValidateConstraint(data, map[string]interface{}{"not_exist": "x"})
		if ok {
			t.Error("expected constraint to fail on missing field")
		}
	})

	t.Run("mismatch fails", func(t *testing.T) {
		ok := ValidateConstraint(data, map[string]interface{}{"status": "disabled"})
		if ok {
			t.Error("expected constraint to fail on mismatch")
		}
	})

	t.Run("multiple conditions all must pass", func(t *testing.T) {
		ok := ValidateConstraint(data, map[string]interface{}{
			"status": "active",
			"amount": 100,
		})
		if !ok {
			t.Error("expected all conditions to pass")
		}
	})
}

// TestValidateConstraintOperators 验证带操作符的约束（委托 model 实现）
func TestValidateConstraintOperators(t *testing.T) {
	data := map[string]interface{}{
		"status": "active",
		"amount": 100,
		"tags":   "a,b,c",
	}

	t.Run("__gt", func(t *testing.T) {
		if !ValidateConstraint(data, map[string]interface{}{"amount": map[string]interface{}{"__gt": 50}}) {
			t.Error("__gt should match")
		}
		if ValidateConstraint(data, map[string]interface{}{"amount": map[string]interface{}{"__gt": 200}}) {
			t.Error("__gt should not match")
		}
	})

	t.Run("__in", func(t *testing.T) {
		if !ValidateConstraint(data, map[string]interface{}{"status": map[string]interface{}{"__in": []interface{}{"active", "disabled"}}}) {
			t.Error("__in should match")
		}
	})

	t.Run("__like", func(t *testing.T) {
		if !ValidateConstraint(data, map[string]interface{}{"tags": map[string]interface{}{"__like": "b"}}) {
			t.Error("__like should match substring")
		}
	})

	t.Run("unknown operator rejected", func(t *testing.T) {
		if ValidateConstraint(data, map[string]interface{}{"amount": map[string]interface{}{"__bogus": 100}}) {
			t.Error("unknown operator should be rejected")
		}
	})
}

// openUtilsTestDB 打开内存 sqlite 供 utils 层级测试使用
func openUtilsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:utils_" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&loc=auto"
	db, err := gorm.Open(
		sqlite.Open(dsn),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Tenant{}); err != nil {
		t.Fatalf("migrate tenant: %v", err)
	}
	return db
}

// TestUpdateTenantPath 验证租户路径/层级更新
func TestUpdateTenantPath(t *testing.T) {
	db := openUtilsTestDB(t)

	// 根租户（无父）
	root := &model.Tenant{ID: 1, Name: "root"}
	if err := db.Create(root).Error; err != nil {
		t.Fatalf("create root: %v", err)
	}
	if err := UpdateTenantPath(db, root); err != nil {
		t.Fatalf("update root path: %v", err)
	}
	if root.Path != "/1/" || root.Level != 1 {
		t.Errorf("root path = %q level = %d, want /1/ and 1", root.Path, root.Level)
	}

	// 子租户
	pid := uint64(1)
	child := &model.Tenant{ID: 2, Name: "child", ParentID: &pid}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}
	if err := UpdateTenantPath(db, child); err != nil {
		t.Fatalf("update child path: %v", err)
	}
	if child.Path != "/1/2/" || child.Level != 2 {
		t.Errorf("child path = %q level = %d, want /1/2/ and 2", child.Path, child.Level)
	}

	// 父租户不存在 → 错误
	orphanPid := uint64(999)
	orphan := &model.Tenant{ID: 3, Name: "orphan", ParentID: &orphanPid}
	if err := UpdateTenantPath(db, orphan); err == nil {
		t.Error("orphan tenant path update should fail when parent missing")
	}
}

// TestGetTenantChildren 验证获取子租户（含自身，通过 path 前缀）
func TestGetTenantChildren(t *testing.T) {
	db := openUtilsTestDB(t)
	root := &model.Tenant{ID: 1, Name: "root"}
	db.Create(root)
	UpdateTenantPath(db, root)
	pid := uint64(1)
	child := &model.Tenant{ID: 2, Name: "child", ParentID: &pid}
	db.Create(child)
	UpdateTenantPath(db, child)

	children, err := GetTenantChildren(db, 1)
	if err != nil {
		t.Fatalf("get children: %v", err)
	}
	// 通过 path LIKE "/1/%" 会包含自身
	if len(children) != 2 {
		t.Errorf("expected 2 tenants (self+child), got %d: %+v", len(children), children)
	}

	// 不存在的租户
	if _, err := GetTenantChildren(db, 999); err == nil {
		t.Error("GetTenantChildren for missing tenant should fail")
	}
}

// seedTreeTenants 播种三层租户结构：根 1 → 子 2 → 孙 3
func seedTreeTenants(t *testing.T, db *gorm.DB) (root, child *model.Tenant) {
	t.Helper()
	root = &model.Tenant{ID: 1, Name: "root"}
	if err := db.Create(root).Error; err != nil {
		t.Fatalf("create root: %v", err)
	}
	if err := UpdateTenantPath(db, root); err != nil {
		t.Fatalf("update root path: %v", err)
	}
	pid := uint64(1)
	child = &model.Tenant{ID: 2, Name: "child", ParentID: &pid}
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
	return root, child
}

// TestGetTenantTree 验证租户树构建
func TestGetTenantTree(t *testing.T) {
	db := openUtilsTestDB(t)
	_, child := seedTreeTenants(t, db)

	t.Run("all roots", func(t *testing.T) {
		// 全部根租户：根 1 → 子 2 → 孙 3
		tree, err := GetTenantTree(db, nil)
		if err != nil {
			t.Fatalf("get tree all: %v", err)
		}
		if len(tree) != 1 {
			t.Fatalf("expected 1 root, got %d", len(tree))
		}
		node := tree[0]
		if node["id"].(uint64) != 1 {
			t.Errorf("root id = %v, want 1", node["id"])
		}
		children, ok := node["children"].([]map[string]interface{})
		if !ok || len(children) != 1 {
			t.Fatalf("expected 1 child, got %v", node["children"])
		}
		if children[0]["id"].(uint64) != 2 {
			t.Errorf("child id = %v, want 2", children[0]["id"])
		}
		grandChildren, ok := children[0]["children"].([]map[string]interface{})
		if !ok || len(grandChildren) != 1 || grandChildren[0]["id"].(uint64) != 3 {
			t.Errorf("grandchild should be tenant 3, got %v", children[0]["children"])
		}
	})

	t.Run("subtree root", func(t *testing.T) {
		// 以租户 2 为根 → 子树 2 → 孙 3
		childID := child.ID
		tree2, err := GetTenantTree(db, &childID)
		if err != nil {
			t.Fatalf("get tree root: %v", err)
		}
		if len(tree2) != 1 || tree2[0]["id"].(uint64) != 2 {
			t.Errorf("subtree root should be tenant 2, got %+v", tree2)
		}
		if gc, ok := tree2[0]["children"].([]map[string]interface{}); !ok || len(gc) != 1 || gc[0]["id"].(uint64) != 3 {
			t.Errorf("subtree should contain grandchild 3, got %v", tree2[0]["children"])
		}
	})

	t.Run("missing root", func(t *testing.T) {
		missing := uint64(999)
		if _, err := GetTenantTree(db, &missing); err == nil {
			t.Error("GetTenantTree for missing root should fail")
		}
	})
}
