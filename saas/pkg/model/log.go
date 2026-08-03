package model

import "time"

// DataShareLog 数据共享日志（用于审计和同步）
type DataShareLog struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	DataID      uint64    `gorm:"not null;index:idx_sharelog_data" json:"data_id"`
	DataType    string    `gorm:"type:varchar(50);not null;index:idx_sharelog_data" json:"data_type"`
	FromTenant  uint64    `gorm:"not null" json:"from_tenant"`
	ToTenant    uint64    `gorm:"not null" json:"to_tenant"`
	Action      string    `gorm:"type:varchar(20);not null" json:"action"` // grant/revoke/expire
	AccessLevel int       `gorm:"not null" json:"access_level"`
	OperatorID  uint64    `gorm:"not null" json:"operator_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableName 指定表名
func (DataShareLog) TableName() string {
	return "saas_data_share_log"
}
