package saas

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LandcLi/landc-go/saas/pkg/model"

	"gorm.io/gorm"
)

// UpdateTenantPath 更新租户路径（在创建/更新租户后调用）
func UpdateTenantPath(db *gorm.DB, tenant *model.Tenant) error {
	if tenant.ParentID == nil {
		// 根租户
		tenant.Path = fmt.Sprintf("/%d/", tenant.ID)
		tenant.Level = 1
	} else {
		// 子租户：获取父租户的路径
		var parent model.Tenant
		if err := db.First(&parent, *tenant.ParentID).Error; err != nil {
			return err
		}
		tenant.Path = fmt.Sprintf("%s%d/", parent.Path, tenant.ID)
		tenant.Level = parent.Level + 1
	}

	return db.Save(tenant).Error
}

// GetTenantChildren 获取所有子租户（递归）
func GetTenantChildren(db *gorm.DB, tenantID uint64) ([]model.Tenant, error) {
	var tenant model.Tenant
	if err := db.First(&tenant, tenantID).Error; err != nil {
		return nil, err
	}

	var children []model.Tenant
	if err := db.Where("path LIKE ?", tenant.Path+"%").Find(&children).Error; err != nil {
		return nil, err
	}

	return children, nil
}

// GetTenantTree 获取租户树结构
func GetTenantTree(db *gorm.DB, rootID *uint64) ([]map[string]interface{}, error) {
	var tenants []model.Tenant
	query := db.Order("level ASC, id ASC")

	if rootID == nil {
		// 获取所有根租户
		query = query.Where("parent_id IS NULL")
	} else {
		// 获取指定租户及其所有子租户
		var root model.Tenant
		if err := db.First(&root, *rootID).Error; err != nil {
			return nil, err
		}
		query = query.Where("path LIKE ?", root.Path+"%")
	}

	query.Find(&tenants)

	// 构建树结构
	return buildTree(tenants, nil), nil
}

// buildTree 构建树结构（辅助函数）
func buildTree(tenants []model.Tenant, parentID *uint64) []map[string]interface{} {
	var tree []map[string]interface{}

	for _, tenant := range tenants {
		var currentParentID *uint64
		if tenant.ParentID != nil {
			currentParentID = tenant.ParentID
		}

		if equals(currentParentID, parentID) {
			node := map[string]interface{}{
				"id":       tenant.ID,
				"name":     tenant.Name,
				"level":    tenant.Level,
				"children": buildTree(tenants, &tenant.ID),
			}
			tree = append(tree, node)
		}
	}

	return tree
}

// equals 比较两个指针是否相等（辅助函数）
func equals(a, b *uint64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// ParseConstraint 解析约束条件
func ParseConstraint(constraintStr string) (map[string]interface{}, error) {
	if constraintStr == "" {
		return nil, nil
	}

	var constraint map[string]interface{}
	err := json.Unmarshal([]byte(constraintStr), &constraint)
	return constraint, err
}

// ValidateConstraint 验证数据是否满足约束条件
func ValidateConstraint(data map[string]interface{}, constraint map[string]interface{}) bool {
	for key, expected := range constraint {
		actual, ok := data[key]
		if !ok {
			return false
		}

		if !compareValues(actual, expected) {
			return false
		}
	}

	return true
}

// compareValues 比较值
// 支持操作符语法：{"__eq": v} | {"__ne": v} | {"__gt": v} | {"__gte": v} |
// {"__lt": v} | {"__lte": v} | {"__in": [v1, v2, ...]} | {"__like": "prefix"}
// 无操作符时使用等值比较
func compareValues(actual, expected interface{}) bool {
	if opMap, ok := expected.(map[string]interface{}); ok {
		if v, has := opMap["__eq"]; has {
			return actual == v
		}
		if v, has := opMap["__ne"]; has {
			return actual != v
		}
		if v, has := opMap["__in"]; has {
			return valueIn(actual, v)
		}
		if v, has := opMap["__like"]; has {
			str, ok := toString(actual)
			if !ok {
				return false
			}
			return strings.Contains(str, v.(string))
		}
		if v, has := opMap["__gt"]; has {
			return compareNumeric(actual, v) > 0
		}
		if v, has := opMap["__gte"]; has {
			return compareNumeric(actual, v) >= 0
		}
		if v, has := opMap["__lt"]; has {
			return compareNumeric(actual, v) < 0
		}
		if v, has := opMap["__lte"]; has {
			return compareNumeric(actual, v) <= 0
		}
		// 未知操作符：保守拒绝
		return false
	}

	return actual == expected
}

// valueIn 判断 actual 是否在列表中
func valueIn(actual, list interface{}) bool {
	switch l := list.(type) {
	case []interface{}:
		for _, item := range l {
			if actual == item {
				return true
			}
		}
	case []string:
		for _, item := range l {
			if actual == item {
				return true
			}
		}
	case []int:
		for _, item := range l {
			if actual == item {
				return true
			}
		}
	case []float64:
		for _, item := range l {
			if actual == item {
				return true
			}
		}
	}
	return false
}

// compareNumeric 数值比较（-1, 0, 1）
func compareNumeric(actual, expected interface{}) int {
	a, ok := toFloat64(actual)
	if !ok {
		return -2 // 无法转换
	}
	b, ok := toFloat64(expected)
	if !ok {
		return 2
	}
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// toFloat64 将数字类型转为 float64
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

// toString 将字符串类型取出
func toString(v interface{}) (string, bool) {
	s, ok := v.(string)
	return s, ok
}
