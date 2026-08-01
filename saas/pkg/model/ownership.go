package model

import "time"

// DataOwnership 数据归属表（核心表）
// 记录数据的拥有者，一条数据只能有一个拥有者
type DataOwnership struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	DataID    uint64    `gorm:"not null;index:idx_ownership_data" json:"data_id"`       // 数据ID
	DataType  string    `gorm:"type:varchar(50);not null;index:idx_ownership_data" json:"data_type"` // 数据类型（表名）
	OwnerID   uint64    `gorm:"not null;index:idx_owner" json:"owner_id"`    // 拥有者租户ID
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (DataOwnership) TableName() string {
	return "saas_data_ownership"
}

// 复合唯一索引：同一数据的同一类型只能有一个拥有者
// 如果需要多拥有者，删除此索引，改用 DataAccess 表
