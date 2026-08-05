package db

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

// Transaction 执行事务。
// 连接基于 ctx 中的资源作用域解析（db.GetDBFrom）：无作用域时回退全局连接。
func Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	db := GetDBFrom(ctx)
	if db == nil {
		return gorm.ErrInvalidDB
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx)
	})
}

// GetTx 从 context 获取事务实例；没有事务时返回基于作用域解析的连接（无作用域回退全局）。
func GetTx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return GetDBFrom(ctx)
}

// Paginate 分页查询辅助
func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 10
		}
		if pageSize > 1000 {
			pageSize = 1000
		}
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}
