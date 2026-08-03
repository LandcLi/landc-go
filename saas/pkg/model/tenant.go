package model

import (
	"time"

	"gorm.io/gorm"
)

// Tenant 租户表（支持层级）
type Tenant struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"type:varchar(100);not null;index:idx_name" json:"name"`
	ParentID  *uint64   `gorm:"index:idx_parent" json:"parent_id"`            // 父租户
	Path      string    `gorm:"type:varchar(500);index:idx_path" json:"path"` // 物化路径: /1/3/5/
	Level     int       `gorm:"default:1" json:"level"`
	Status    int       `gorm:"not null;default:1" json:"status"` // 1=正常 2=禁用
	Config    string    `gorm:"type:json" json:"config"`          // 租户配置
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Tenant) TableName() string {
	return "saas_tenants"
}

// IsRoot 是否根租户
func (t *Tenant) IsRoot() bool {
	return t.ParentID == nil
}

// HasChildren 是否有子租户
func (t *Tenant) HasChildren(db *gorm.DB) bool {
	var count int64
	db.Model(&Tenant{}).Where("parent_id = ?", t.ID).Count(&count)
	return count > 0
}
