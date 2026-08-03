// saas 多租户模块使用示例
//
// 生产环境请使用 MySQL/PostgreSQL；此处用 SQLite 内存库便于直接运行演示。
// 运行：go run ./examples
package main

import (
	"context"
	"fmt"
	"time"

	saaspkg "github.com/LandcLi/landc-go/saas/pkg"
	"github.com/LandcLi/landc-go/saas/pkg/model"
	sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Product 示例业务表
type Product struct {
	ID     uint64  `gorm:"primaryKey" json:"id"`
	Name   string  `gorm:"type:varchar(100)" json:"name"`
	Price  float64 `json:"price"`
	Status string  `gorm:"type:varchar(20)" json:"status"`
}

func (Product) TableName() string { return "products" }

func mustDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:saas_demo?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		panic(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	if err := saaspkg.AutoMigrate(db); err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&Product{}); err != nil {
		panic(err)
	}
	return db
}

func main() {
	db := mustDB()
	m := saaspkg.NewManager(db, saaspkg.WithConfig(saaspkg.Config{
		EnableHierarchy:  true,
		EnableConstraint: true,
		CleanupInterval:  time.Hour,
	}))

	// 1. 创建租户（直接 GORM 创建 + 维护物化路径）
	t1 := &model.Tenant{Name: "租户A"}
	if err := db.Create(t1).Error; err != nil {
		panic(err)
	}
	if err := saaspkg.UpdateTenantPath(db, t1); err != nil {
		panic(err)
	}
	t2 := &model.Tenant{Name: "租户B"}
	if err := db.Create(t2).Error; err != nil {
		panic(err)
	}
	if err := saaspkg.UpdateTenantPath(db, t2); err != nil {
		panic(err)
	}
	fmt.Printf("created tenants: A(id=%d) B(id=%d)\n", t1.ID, t2.ID)

	// 2. 准备业务数据（租户A 拥有两个商品）
	if err := db.Create(&[]Product{
		{ID: 1, Name: "旗舰手机", Price: 4999, Status: "active"},
		{ID: 2, Name: "入门耳机", Price: 199, Status: "active"},
		{ID: 3, Name: "下架商品", Price: 99, Status: "disabled"},
	}).Error; err != nil {
		panic(err)
	}

	ctxA := saaspkg.WithTenant(context.Background(), t1.ID)
	tx := db.Begin()
	for _, id := range []uint64{1, 2, 3} {
		if err := m.CreateData(ctxA, tx, "products", id); err != nil {
			panic(err)
		}
	}
	tx.Commit()

	// 3. 租户A 把"商品 2"共享给租户B，并附加约束：仅 price < 500 的数据可访问
	tx = db.Begin()
	if err := m.ShareData(ctxA, tx, "products", 2, t2.ID, saaspkg.AccessRead, nil, map[string]interface{}{
		"price": map[string]interface{}{"__lt": 500},
	}); err != nil {
		panic(err)
	}
	tx.Commit()

	// 4. 租户A 视角：能看到自己全部 3 个商品
	ctxA2 := saaspkg.WithTenant(context.Background(), t1.ID)
	var aIDs []uint64
	if err := db.Model(&Product{}).Scopes(m.TenantScope(ctxA2, "products")).Pluck("id", &aIDs).Error; err != nil {
		panic(err)
	}
	fmt.Println("租户A 可见商品:", aIDs)

	// 5. 租户B 视角（带约束）：仅满足 price < 500 的共享商品（商品 2，price=199）
	ctxB := saaspkg.WithTenant(context.Background(), t2.ID)
	var bIDs []uint64
	if err := db.Model(&Product{}).Scopes(m.TenantScopeWithConstraint(ctxB, "products")).Pluck("id", &bIDs).Error; err != nil {
		panic(err)
	}
	fmt.Println("租户B 可见商品(带约束):", bIDs)

	// 6. 撤销共享后，租户B 不再可见
	tx = db.Begin()
	if err := m.RevokeAccess(ctxA2, tx, "products", 2, t2.ID); err != nil {
		panic(err)
	}
	tx.Commit()

	bIDs = nil
	if err := db.Model(&Product{}).Scopes(m.TenantScopeWithConstraint(ctxB, "products")).Pluck("id", &bIDs).Error; err != nil {
		panic(err)
	}
	fmt.Println("撤销共享后 租户B 可见商品:", bIDs)
}
