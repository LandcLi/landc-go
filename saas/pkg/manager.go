package saas

import (
	"context"
	"fmt"
	"github.com/LandcLi/landc-go/saas/pkg/model"
	"time"

	"gorm.io/gorm"
)

// Config SaaS配置
type Config struct {
	EnableHierarchy  bool          // 是否启用层级
	EnableConstraint bool          // 是否启用约束
	CleanupInterval time.Duration // 清理过期访问记录的间隔（0=不清理）
}

// Manager SaaS管理器（无状态，并发安全）
type Manager struct {
	db     *gorm.DB
	config Config
}

// NewManager 创建SaaS管理器
func NewManager(db *gorm.DB, opts ...Option) *Manager {
	m := &Manager{
		db: db,
		config: Config{
			EnableHierarchy: true,
			CleanupInterval: 1 * time.Hour,
		},
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Option 配置选项
type Option func(*Manager)

// WithConfig 设置配置
func WithConfig(config Config) Option {
	return func(m *Manager) {
		m.config = config
	}
}

// getTenantFromContext 从上下文获取租户ID（辅助方法）
func (m *Manager) getTenantFromContext(ctx context.Context) (uint64, error) {
	tenantID, ok := GetTenantFromContext(ctx)
	if !ok {
		return 0, fmt.Errorf("上下文中未找到租户信息，请使用 saas.WithTenant(ctx, tenantID)")
	}
	return tenantID, nil
}

// AccessLevel 访问级别常量
const (
	AccessRead  = 1 // 只读
	AccessWrite = 2 // 读写
	AccessAdmin = 3 // 管理
)

// GrantType 授权类型常量
const (
	GrantDirect  = 1 // 直接授权
	GrantInherit = 2 // 继承
	GrantProxy   = 3 // 代理
)

// Event 类型常量
const (
	EventDataCreated          = "data_created"
	EventDataShared           = "data_shared"
	EventAccessRevoked        = "access_revoked"
	EventOwnershipTransferred = "ownership_transferred"
)

// AutoMigrate 自动迁移SaaS相关表
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Tenant{},
		&model.DataOwnership{},
		&model.DataAccess{},
		&model.DataShareLog{},
	)
}
