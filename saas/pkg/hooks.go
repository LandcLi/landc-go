// Package saas 提供多租户数据隔离能力
//
// 核心概念：
//   - 数据归属（DataOwnership）：记录数据的拥有者（一对多）
//   - 数据访问（DataAccess）：记录哪些租户可访问数据（多对多）
//   - 租户上下文（context.Context）：通过 context 传递当前租户信息
//
// 快速开始：
//  1. 创建 Manager：manager := saas.NewManager(db)
//  2. 设置租户上下文：ctx := saas.WithTenant(context.Background(), tenantID)
//  3. 查询数据：db.WithContext(ctx).Scopes(manager.TenantScope(ctx, "orders")).Find(&orders)
//  4. 创建数据：db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
//     tx.Create(&order)
//     return manager.CreateData(ctx, tx, "orders", order.ID)
//     })
//
// 设计理念：
//   - 零侵入：不修改业务表，通过中间关系表实现隔离
//   - 并发安全：使用 context.Context 传递租户信息，无全局状态
//   - 纯数据库：不依赖缓存或消息队列，简单可靠
package saas

// 此文件 intentionally 留空，所有核心功能在 manager.go、scope.go、operations.go 中
