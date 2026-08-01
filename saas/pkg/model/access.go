package model

import "time"

// DataAccess 数据访问表（支持共享、多对多）
// 记录所有可访问某数据的租户
type DataAccess struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	DataID      uint64    `gorm:"not null;index:idx_access_data" json:"data_id"`       // 数据ID
	DataType    string    `gorm:"type:varchar(50);not null;index:idx_access_data" json:"data_type"` // 数据类型
	TenantID    uint64    `gorm:"not null;index:idx_tenant" json:"tenant_id"`   // 可访问租户ID
	AccessLevel int       `gorm:"not null;default:1" json:"access_level"`        // 1=读 2=写 3=管理
	GrantType   int       `gorm:"not null;default:1" json:"grant_type"`          // 1=直接授权 2=继承 3=代理
	GrantedBy  uint64    `gorm:"not null" json:"granted_by"`                    // 授权人租户ID
	Constraints string    `gorm:"type:json" json:"constraints"`                  // 访问约束（JSON）
	ExpireAt    *time.Time `json:"expire_at"`                                   // 过期时间（NULL=永不过期）
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (DataAccess) TableName() string {
	return "saas_data_access"
}

// 复合唯一索引：同一租户对同一数据的同一类型只能有一条记录

// IsExpired 是否过期
func (d *DataAccess) IsExpired() bool {
	if d.ExpireAt == nil {
		return false
	}
	return d.ExpireAt.Before(time.Now())
}

// CheckConstraint 检查约束条件
func (d *DataAccess) CheckConstraint(data map[string]interface{}) bool {
	if d.Constraints == "" {
		return true
	}
	
	// TODO: 实现约束检查逻辑
	// 解析 Constraints JSON，与 data 对比
	return true
}
