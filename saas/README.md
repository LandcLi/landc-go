# landc-go/saas

通用多租户 SaaS 插件，为 Go 项目提供零侵入的多租户数据隔离能力。

## 特性

- ✅ **零侵入**：不修改现有业务表，通过中间关系表实现隔离
- ✅ **通用性**：支持任意业务表，通过配置适配
- ✅ **并发安全**：使用 `context.Context` 传递租户信息，无全局状态
- ✅ **纯数据库**：不依赖缓存或消息队列，简单可靠
- ✅ **高扩展**：支持一对多、多对多、层级关系
- ✅ **灵活授权**：支持直接授权、继承、代理、临时授权

## 核心概念

### 1. 数据归属 (DataOwnership)
记录数据的拥有者，一条数据只能有一个拥有者。

### 2. 数据访问 (DataAccess)
记录所有可访问某数据的租户，支持：
- 一对多：一个租户拥有，多个租户访问
- 多对多：多个租户都能访问同一数据
- 层级继承：子租户继承父租户的数据权限
- 临时授权：支持过期时间

### 3. 租户层级 (Tenant)
支持租户层级关系，使用物化路径实现高效查询。

## 快速开始

### 安装

```go
import "landc-go/saas"
```

### 初始化

```go
package main

import (
    "context"
    "gorm.io/gorm"
    "landc-go/saas"
)

func main() {
    // 1. 初始化数据库
    db, _ := gorm.Open(...)

    // 2. 自动迁移（创建 SaaS 相关表）
    saas.AutoMigrate(db)

    // 3. 创建 SaaS 管理器
    manager := saas.NewManager(db, saas.WithConfig(saas.Config{
        EnableHierarchy:  true,
        EnableConstraint: true,
    }))
}
```

### 基本使用

#### 1. 设置租户上下文

```go
// 在请求入口（如中间件）设置租户信息
ctx := saas.WithTenant(context.Background(), 1001)

// 传递给后续的数据库操作
db.WithContext(ctx).Find(&orders)
```

#### 2. 查询数据（自动租户隔离）

```go
// 查询当前租户可访问的订单（自己拥有的 + 共享给自己的）
var orders []Order
db.WithContext(ctx).Scopes(manager.TenantScope(ctx, "orders")).Find(&orders)

// 只查询自己拥有的订单
db.WithContext(ctx).Scopes(manager.TenantScopeWithOwn(ctx, "orders")).Find(&orders)

// 只查询共享给自己的订单
db.WithContext(ctx).Scopes(manager.TenantScopeWithAccess(ctx, "orders")).Find(&orders)

// 查询自己及子租户的数据（需启用层级）
db.WithContext(ctx).Scopes(manager.TenantScopeWithChildren(ctx, "orders")).Find(&orders)
```

#### 3. 创建数据

```go
err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    // 1. 创建业务数据
    order := &Order{
        OrderNo: "ORD-001",
        Amount:  100.00,
    }
    if err := tx.Create(order).Error; err != nil {
        return err
    }

    // 2. 创建租户关系（自动维护归属和访问记录）
    if err := manager.CreateData(ctx, tx, "orders", order.ID); err != nil {
        return err
    }

    return nil
})
```

#### 4. 共享数据

```go
// 将订单共享给租户 2001，只读权限，7天后过期
expireAt := time.Now().Add(7 * 24 * time.Hour)
err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    return manager.ShareData(ctx, tx, "orders", orderID, 2001, saas.AccessRead, &expireAt, nil)
}).Error
```

#### 5. 撤销访问权限

```go
err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    return manager.RevokeAccess(ctx, tx, "orders", orderID, 2001)
}).Error
```

#### 6. 转移数据归属

```go
err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    return manager.TransferOwnership(ctx, tx, "orders", orderID, 2001)
}).Error
```

## 数据库表结构

### 1. saas_tenants（租户表）

```sql
CREATE TABLE saas_tenants (
    id         BIGINT PRIMARY KEY AUTO_INCREMENT,
    name       VARCHAR(100) NOT NULL,
    parent_id  BIGINT,
    path       VARCHAR(500),
    level      INT DEFAULT 1,
    status     INT DEFAULT 1,
    config     JSON,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    INDEX idx_parent (parent_id),
    INDEX idx_path (path)
);
```

### 2. saas_data_ownership（数据归属表）

```sql
CREATE TABLE saas_data_ownership (
    id         BIGINT PRIMARY KEY AUTO_INCREMENT,
    data_id    BIGINT NOT NULL,
    data_type  VARCHAR(50) NOT NULL,
    owner_id   BIGINT NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    INDEX idx_data (data_type, data_id),
    INDEX idx_owner (owner_id),
    UNIQUE KEY uk_data_type_owner (data_type, data_id)
);
```

### 3. saas_data_access（数据访问表）

```sql
CREATE TABLE saas_data_access (
    id            BIGINT PRIMARY KEY AUTO_INCREMENT,
    data_id       BIGINT NOT NULL,
    data_type     VARCHAR(50) NOT NULL,
    tenant_id     BIGINT NOT NULL,
    access_level  INT DEFAULT 1,
    grant_type    INT DEFAULT 1,
    granted_by    BIGINT NOT NULL,
    constraints   JSON,
    expire_at     TIMESTAMP,
    created_at    TIMESTAMP,
    updated_at    TIMESTAMP,
    INDEX idx_data (data_type, data_id),
    INDEX idx_tenant (tenant_id),
    UNIQUE KEY uk_data_type_tenant (data_type, data_id, tenant_id)
);
```

### 4. saas_data_share_log（审计日志表）

```sql
CREATE TABLE saas_data_share_log (
    id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    data_id     BIGINT NOT NULL,
    data_type   VARCHAR(50) NOT NULL,
    from_tenant BIGINT NOT NULL,
    to_tenant   BIGINT NOT NULL,
    action      VARCHAR(20) NOT NULL,
    access_level INT,
    operator_id BIGINT NOT NULL,
    created_at  TIMESTAMP,
    INDEX idx_data (data_type, data_id)
);
```

## 性能优化

### 1. 索引优化
确保以下索引已创建：
- `saas_data_ownership`: INDEX `idx_owner` (`owner_id`)
- `saas_data_access`: INDEX `idx_tenant` (`tenant_id`)
- `saas_data_access`: INDEX `idx_data` (`data_type`, `data_id`)

### 2. 分页查询
对于大数据量，使用分页：

```go
dataIDs, total, err := manager.ListTenantData(ctx, "orders", tenantID, 1, 20)
```

### 3. 定期清理过期数据

```go
// 在定时任务中调用
func CleanupExpiredAccess(db *gorm.DB) error {
    return db.Where("expire_at < NOW()").Delete(&model.DataAccess{}).Error
}
```

## 最佳实践

### 1. 在中间件中设置租户上下文

```go
func TenantMiddleware(manager *saas.Manager) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从 JWT/Header/Token 中获取租户ID
        tenantID := getTenantIDFromToken(c)

        // 设置到上下文
        ctx := saas.WithTenant(c.Request.Context(), tenantID)
        c.Request = c.Request.WithContext(ctx)

        c.Next()
    }
}
```

### 2. 使用事务保证一致性

```go
err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    // 所有操作在同一个事务中
    tx.Create(&order)
    manager.CreateData(ctx, tx, "orders", order.ID)
    return nil
})
```

### 3. 批量操作优化

```go
// 批量创建数据归属关系
err := saas.BatchCreateData(db, "orders", tenantID, []uint64{1, 2, 3, 4, 5})
```

## 常见问题

### Q: 为什么不使用缓存？
A: 作为通用插件，我们不希望强制依赖 Redis 等外部组件。如果您的项目需要缓存，可以在业务层自行实现。

### Q: 如何支持多租户管理后台？
A: 可以使用 `GetTenantTree` 获取租户树，使用 `GetTenantDataStats` 获取统计信息。

### Q: 如何清理过期数据？
A: 使用 `CleanupExpiredAccess(db)` 函数，建议在定时任务中调用。

## License

MIT
