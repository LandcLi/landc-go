package saas

import (
	"fmt"

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

	if rootID == nil {
		// 获取全部租户（buildTree 以 parent_id IS NULL 为根递归构建）
		if err := db.Model(&model.Tenant{}).
			Order("level ASC, id ASC").
			Find(&tenants).Error; err != nil {
			return nil, err
		}
		return buildTree(tenants, nil), nil
	}

	// 获取指定租户及其所有子租户
	var root model.Tenant
	if err := db.First(&root, *rootID).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&model.Tenant{}).
		Order("level ASC, id ASC").
		Where("path LIKE ?", root.Path+"%").
		Find(&tenants).Error; err != nil {
		return nil, err
	}

	// 子树场景：顶层节点为 root 自身，其父不在结果集内，
	// 因此以 root.ParentID 作为顶层匹配（只有 root 命中），递归构建子树
	return buildTree(tenants, root.ParentID), nil
}

// buildTree 构建树结构（辅助函数）
func buildTree(tenants []model.Tenant, parentID *uint64) []map[string]interface{} {
	var tree []map[string]interface{}

	for i := range tenants {
		tenant := &tenants[i]
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

// ParseConstraint 解析约束条件（委托 model 包实现，保持 API 兼容）
func ParseConstraint(constraintStr string) (map[string]interface{}, error) {
	return model.ParseConstraint(constraintStr)
}

// ValidateConstraint 验证数据是否满足约束条件（委托 model 包实现，保持 API 兼容）
func ValidateConstraint(data, constraint map[string]interface{}) bool {
	return model.ValidateConstraint(data, constraint)
}
