package saas

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/LandcLi/landc-go/saas/pkg/model"
	"time"

	"gorm.io/gorm"
)

// CreateData 创建数据（自动维护归属关系）
func (m *Manager) CreateData(ctx context.Context, tx *gorm.DB, dataType string, dataID uint64) error {
	tenantID, err := m.getTenantFromContext(ctx)
	if err != nil {
		return err
	}

	// 1. 创建归属关系
	ownership := model.DataOwnership{
		DataID:    dataID,
		DataType:  dataType,
		OwnerID:   tenantID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := tx.Create(&ownership).Error; err != nil {
		return err
	}

	// 2. 自动创建访问记录（拥有者默认有管理权限）
	access := model.DataAccess{
		DataID:      dataID,
		DataType:    dataType,
		TenantID:    tenantID,
		AccessLevel: AccessAdmin,
		GrantType:   GrantDirect,
		GrantedBy:   tenantID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return tx.Create(&access).Error
}

// ShareData 共享数据给租户
func (m *Manager) ShareData(ctx context.Context, tx *gorm.DB, dataType string, dataID, toTenantID uint64,
	accessLevel int, expireAt *time.Time, constraints map[string]interface{}) error {
	tenantID, err := m.getTenantFromContext(ctx)
	if err != nil {
		return err
	}

	// 1. 检查权限（只有拥有者才能共享）
	var ownership model.DataOwnership
	err = tx.Where("data_type = ? AND data_id = ? AND owner_id = ?",
		dataType, dataID, tenantID).First(&ownership).Error
	if err != nil {
		return fmt.Errorf("无权限共享此数据或无此数据")
	}

	// 2. 创建或更新访问记录
	access := model.DataAccess{
		DataID:      dataID,
		DataType:    dataType,
		TenantID:    toTenantID,
		AccessLevel: accessLevel,
		GrantType:   GrantDirect,
		GrantedBy:   tenantID,
		ExpireAt:    expireAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if constraints != nil {
		data, _ := json.Marshal(constraints)
		access.Constraints = string(data)
	}

	// UPSERT（更新或插入）
	tx.Where("data_type = ? AND data_id = ? AND tenant_id = ?",
		dataType, dataID, toTenantID).
		Assign(access).
		FirstOrCreate(&access)

	// 3. 记录日志
	log := model.DataShareLog{
		DataID:      dataID,
		DataType:    dataType,
		FromTenant:  tenantID,
		ToTenant:    toTenantID,
		Action:      "grant",
		AccessLevel: accessLevel,
		OperatorID:  tenantID,
		CreatedAt:   time.Now(),
	}

	return tx.Create(&log).Error
}

// RevokeAccess 撤销访问权限
func (m *Manager) RevokeAccess(ctx context.Context, tx *gorm.DB, dataType string, dataID, fromTenantID uint64) error {
	tenantID, err := m.getTenantFromContext(ctx)
	if err != nil {
		return err
	}

	// 1. 检查权限（只有拥有者才能撤销共享，防止越权撤销他人共享）
	var ownership model.DataOwnership
	err = tx.Where("data_type = ? AND data_id = ? AND owner_id = ?",
		dataType, dataID, tenantID).First(&ownership).Error
	if err != nil {
		return fmt.Errorf("无权限撤销此数据的共享")
	}

	// 2. 删除访问记录
	err = tx.Where("data_type = ? AND data_id = ? AND tenant_id = ?",
		dataType, dataID, fromTenantID).Delete(&model.DataAccess{}).Error
	if err != nil {
		return err
	}

	// 2. 记录日志
	log := model.DataShareLog{
		DataID:      dataID,
		DataType:    dataType,
		FromTenant:  tenantID,
		ToTenant:    fromTenantID,
		Action:      "revoke",
		AccessLevel: 0,
		OperatorID:  tenantID,
		CreatedAt:   time.Now(),
	}

	return tx.Create(&log).Error
}

// TransferOwnership 转移数据归属
func (m *Manager) TransferOwnership(ctx context.Context, tx *gorm.DB, dataType string, dataID, newOwnerID uint64) error {
	tenantID, err := m.getTenantFromContext(ctx)
	if err != nil {
		return err
	}

	// 1. 检查权限
	var ownership model.DataOwnership
	err = tx.Where("data_type = ? AND data_id = ? AND owner_id = ?",
		dataType, dataID, tenantID).First(&ownership).Error
	if err != nil {
		return fmt.Errorf("无权限转移此数据")
	}

	// 2. 更新归属
	ownership.OwnerID = newOwnerID
	ownership.UpdatedAt = time.Now()
	if err := tx.Save(&ownership).Error; err != nil {
		return err
	}

	// 3. 删除原拥有者的访问记录（可选）
	tx.Where("data_type = ? AND data_id = ? AND tenant_id = ?",
		dataType, dataID, tenantID).Delete(&model.DataAccess{})

	// 4. 为新拥有者创建访问记录
	newAccess := model.DataAccess{
		DataID:      dataID,
		DataType:    dataType,
		TenantID:    newOwnerID,
		AccessLevel: AccessAdmin,
		GrantType:   GrantDirect,
		GrantedBy:   tenantID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return tx.Create(&newAccess).Error
}

// DeleteData 删除数据（同时删除关系）
func (m *Manager) DeleteData(ctx context.Context, tx *gorm.DB, dataType string, dataID uint64) error {
	tenantID, err := m.getTenantFromContext(ctx)
	if err != nil {
		return err
	}

	// 1. 检查权限
	var ownership model.DataOwnership
	err = tx.Where("data_type = ? AND data_id = ? AND owner_id = ?",
		dataType, dataID, tenantID).First(&ownership).Error
	if err != nil {
		return fmt.Errorf("无权限删除此数据")
	}

	// 2. 删除关系记录
	tx.Where("data_type = ? AND data_id = ?", dataType, dataID).Delete(&model.DataOwnership{})
	tx.Where("data_type = ? AND data_id = ?", dataType, dataID).Delete(&model.DataAccess{})

	return nil
}

// GetDataOwner 获取数据的拥有者
func (m *Manager) GetDataOwner(ctx context.Context, dataType string, dataID uint64) (uint64, error) {
	var ownership model.DataOwnership
	err := m.db.Where("data_type = ? AND data_id = ?", dataType, dataID).First(&ownership).Error
	if err != nil {
		return 0, err
	}
	return ownership.OwnerID, nil
}

// GetDataAccessors 获取可访问某数据的租户列表
func (m *Manager) GetDataAccessors(ctx context.Context, dataType string, dataID uint64) ([]model.DataAccess, error) {
	var accesses []model.DataAccess
	err := m.db.Where("data_type = ? AND data_id = ? AND (expire_at IS NULL OR expire_at > ?)",
		dataType, dataID, time.Now()).Find(&accesses).Error
	return accesses, err
}

// ListTenantData 列出租户可访问的所有数据（SQL 层分页，避免全量加载）
// 拥有 + 共享 两个数据集 UNION 去重；表名取自模型常量（非用户输入），
// dataType/tenantID/时间/分页参数均参数化绑定，防 SQL 注入。
func (m *Manager) ListTenantData(ctx context.Context, dataType string, tenantID uint64, page, pageSize int) (dataIDs []uint64, total int64, err error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	ownershipTable := model.DataOwnership{}.TableName()
	accessTable := model.DataAccess{}.TableName()
	unionSub := fmt.Sprintf(
		"(SELECT data_id FROM %s WHERE data_type = ? AND owner_id = ? UNION SELECT data_id FROM %s WHERE data_type = ? AND tenant_id = ? AND (expire_at IS NULL OR expire_at > ?)) AS tenant_data",
		ownershipTable, accessTable,
	)
	args := []interface{}{dataType, tenantID, dataType, tenantID, time.Now()}

	// 1. 总数（UNION 已去重）
	if err := m.db.Session(&gorm.Session{NewDB: true}).
		Raw("SELECT COUNT(*) FROM "+unionSub, args...).
		Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// 2. 分页数据
	if err := m.db.Session(&gorm.Session{NewDB: true}).
		Raw("SELECT data_id FROM "+unionSub+" ORDER BY data_id LIMIT ? OFFSET ?",
			append(args, pageSize, (page-1)*pageSize)...).
		Scan(&dataIDs).Error; err != nil {
		return nil, 0, err
	}

	return dataIDs, total, nil
}

// CleanupExpiredAccess 清理过期的访问记录（定时任务）
func CleanupExpiredAccess(db *gorm.DB) error {
	return db.Where("expire_at < ?", time.Now()).Delete(&model.DataAccess{}).Error
}

// GetTenantDataStats 获取租户数据统计
func GetTenantDataStats(db *gorm.DB, tenantID uint64) (map[string]int64, error) {
	stats := make(map[string]int64)

	// 拥有的数据
	var ownCount int64
	db.Model(&model.DataOwnership{}).Where("owner_id = ?", tenantID).Count(&ownCount)
	stats["owned"] = ownCount

	// 可访问的数据
	var accessCount int64
	db.Model(&model.DataAccess{}).Where("tenant_id = ? AND (expire_at IS NULL OR expire_at > ?)", tenantID, time.Now()).Count(&accessCount)
	stats["accessible"] = accessCount

	// 共享给他人的数据
	var sharedCount int64
	db.Model(&model.DataAccess{}).Where("granted_by = ? AND tenant_id != ?", tenantID, tenantID).Count(&sharedCount)
	stats["shared"] = sharedCount

	return stats, nil
}

// BatchCreateData 批量创建数据归属关系（性能优化）
func BatchCreateData(db *gorm.DB, dataType string, tenantID uint64, dataIDs []uint64) error {
	now := time.Now()

	// 批量插入 DataOwnership
	ownerships := make([]model.DataOwnership, len(dataIDs))
	for i, dataID := range dataIDs {
		ownerships[i] = model.DataOwnership{
			DataID:    dataID,
			DataType:  dataType,
			OwnerID:   tenantID,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	if err := db.CreateInBatches(ownerships, 100).Error; err != nil {
		return err
	}

	// 批量插入 DataAccess
	accesses := make([]model.DataAccess, len(dataIDs))
	for i, dataID := range dataIDs {
		accesses[i] = model.DataAccess{
			DataID:      dataID,
			DataType:    dataType,
			TenantID:    tenantID,
			AccessLevel: AccessAdmin,
			GrantType:   GrantDirect,
			GrantedBy:   tenantID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}

	return db.CreateInBatches(accesses, 100).Error
}
