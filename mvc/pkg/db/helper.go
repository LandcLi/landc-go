package db

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

// Transaction 执行事务
func Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	db := GetDB()
	if db == nil {
		return gorm.ErrInvalidDB
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx)
	})
}

// GetTx 从 context 获取事务实例，如果没有则返回全局 DB
func GetTx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return GetDB()
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
