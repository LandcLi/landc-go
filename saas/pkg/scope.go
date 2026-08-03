package saas

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/LandcLi/landc-go/saas/pkg/model"

	"gorm.io/gorm"
)

// validColumnName 校验约束字段名是否安全（仅字母数字下划线，防止 SQL 注入）
var validColumnName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString

// TenantScope 租户隔离作用域（核心方法）
// 使用方式：db.Scopes(manager.TenantScope(ctx, "orders")).Find(&orders)
func (m *Manager) TenantScope(ctx context.Context, dataType string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		tenantID, err := m.getTenantFromContext(ctx)
		if err != nil {
			// 未设置租户信息，返回空结果
			return db.Where("1 = 0")
		}

		// 查询拥有者是自己
		ownQuery := m.db.Model(&model.DataOwnership{}).
			Select("data_id").
			Where("data_type = ? AND owner_id = ?", dataType, tenantID)

		// 查询共享给自己
		accessQuery := m.db.Model(&model.DataAccess{}).
			Select("data_id").
			Where("data_type = ? AND tenant_id = ? AND (expire_at IS NULL OR expire_at > ?)",
				dataType, tenantID, time.Now())

		// 合并查询（使用UNION）
		unionQuery := m.db.Raw("(? UNION ?)", ownQuery, accessQuery)

		return db.Where("id IN (?)", unionQuery)
	}
}

// TenantScopeWithOwn 只查询自己拥有的数据
func (m *Manager) TenantScopeWithOwn(ctx context.Context, dataType string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		tenantID, err := m.getTenantFromContext(ctx)
		if err != nil {
			return db.Where("1 = 0")
		}

		subQuery := m.db.Model(&model.DataOwnership{}).
			Select("data_id").
			Where("data_type = ? AND owner_id = ?", dataType, tenantID)

		return db.Where("id IN (?)", subQuery)
	}
}

// TenantScopeWithAccess 只查询共享给自己的数据
func (m *Manager) TenantScopeWithAccess(ctx context.Context, dataType string, accessLevel ...int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		tenantID, err := m.getTenantFromContext(ctx)
		if err != nil {
			return db.Where("1 = 0")
		}

		query := m.db.Model(&model.DataAccess{}).
			Select("data_id").
			Where("data_type = ? AND tenant_id = ? AND (expire_at IS NULL OR expire_at > NOW())",
				dataType, tenantID)

		if len(accessLevel) > 0 {
			query = query.Where("access_level IN ?", accessLevel)
		}

		return db.Where("id IN (?)", query)
	}
}

// TenantScopeWithChildren 查询自己及子租户的数据（层级）
func (m *Manager) TenantScopeWithChildren(ctx context.Context, dataType string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		tenantID, err := m.getTenantFromContext(ctx)
		if err != nil {
			return db.Where("1 = 0")
		}

		if !m.config.EnableHierarchy {
			// 未启用层级，只查自己的
			return m.TenantScopeWithOwn(ctx, dataType)(db)
		}

		// 1. 获取当前租户的路径
		var tenant model.Tenant
		if err := m.db.First(&tenant, tenantID).Error; err != nil {
			return db.Where("1 = 0")
		}

		// 2. 查找所有子租户
		var childTenants []model.Tenant
		m.db.Where("path LIKE ?", tenant.Path+"%").Find(&childTenants)

		tenantIDs := make([]uint64, len(childTenants)+1)
		tenantIDs[0] = tenantID
		for i := range childTenants {
			tenantIDs[i+1] = childTenants[i].ID
		}

		// 3. 查询这些租户拥有的数据
		subQuery := m.db.Model(&model.DataOwnership{}).
			Select("data_id").
			Where("data_type = ? AND owner_id IN ?", dataType, tenantIDs)

		return db.Where("id IN (?)", subQuery)
	}
}

// TenantScopeWithConstraint 查询带约束的数据
func (m *Manager) TenantScopeWithConstraint(ctx context.Context, dataType string) func(db *gorm.DB) *gorm.DB { //nolint:gocyclo // 约束操作符分派为线性 switch，拆分收益低
	return func(db *gorm.DB) *gorm.DB {
		tenantID, err := m.getTenantFromContext(ctx)
		if err != nil {
			return db.Where("1 = 0")
		}

		if !m.config.EnableConstraint {
			// 未启用约束，使用普通作用域
			return m.TenantScope(ctx, dataType)(db)
		}

		// 1. 获取当前租户的所有访问记录（带约束）
		//    使用独立查询，避免污染后续链式调用
		var accesses []model.DataAccess
		m.db.Model(&model.DataAccess{}).
			Where("data_type = ? AND tenant_id = ? AND (expire_at IS NULL OR expire_at > ?)",
				dataType, tenantID, time.Now()).
			Find(&accesses)

		// 2. 构建ID列表（需要根据约束过滤）
		validIDs := make([]uint64, 0)
		for i := range accesses {
			access := &accesses[i]
			if access.Constraints == "" {
				// 无约束，直接添加
				validIDs = append(validIDs, access.DataID)
				continue
			}

			// 有约束：查询业务数据并验证约束，仅放行满足约束的记录
			constraint, parseErr := ParseConstraint(access.Constraints)
			if parseErr != nil {
				// 约束格式非法：保守策略，拒绝访问
				continue
			}
			if len(constraint) == 0 {
				validIDs = append(validIDs, access.DataID)
				continue
			}

			// 在目标业务表上执行约束查询（列名白名单校验，值参数化）
			// 注意：每次迭代使用全新 Session，避免链状态污染（表名/条件残留）
			q := m.db.Session(&gorm.Session{NewDB: true}).
				Table(dataType).Select("id").Where("id = ?", access.DataID)
			valid := true
			for key, val := range constraint {
				if !validColumnName(key) {
					// 非法列名：拒绝该条访问
					valid = false
					break
				}
				switch op := val.(type) {
				case map[string]interface{}:
					// 操作符语法：{"__gt": v} | {"__gte": v} | {"__lt": v} | {"__lte": v} | {"__eq": v} | {"__ne": v} | {"__in": []}
					if v, has := op["__gt"]; has {
						q = q.Where(fmt.Sprintf("%s > ?", key), v)
					} else if v, has := op["__gte"]; has {
						q = q.Where(fmt.Sprintf("%s >= ?", key), v)
					} else if v, has := op["__lt"]; has {
						q = q.Where(fmt.Sprintf("%s < ?", key), v)
					} else if v, has := op["__lte"]; has {
						q = q.Where(fmt.Sprintf("%s <= ?", key), v)
					} else if v, has := op["__eq"]; has {
						q = q.Where(fmt.Sprintf("%s = ?", key), v)
					} else if v, has := op["__ne"]; has {
						q = q.Where(fmt.Sprintf("%s != ?", key), v)
					} else if v, has := op["__in"]; has {
						q = q.Where(fmt.Sprintf("%s IN ?", key), v)
					} else {
						// 未知操作符：拒绝
						valid = false
					}
				default:
					q = q.Where(fmt.Sprintf("%s = ?", key), val)
				}
			}
			if !valid {
				continue
			}

			var cnt int64
			if err := q.Count(&cnt).Error; err != nil {
				// 查询失败：保守策略，拒绝访问
				continue
			}
			if cnt > 0 {
				validIDs = append(validIDs, access.DataID)
			}
		}

		if len(validIDs) == 0 {
			// 无数据可访问
			return db.Where("1 = 0")
		}

		return db.Where("id IN ?", validIDs)
	}
}
