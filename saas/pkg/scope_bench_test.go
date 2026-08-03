package saas

import (
	"context"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/saas/pkg/model"
	sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func BenchmarkTenantScopeQuery(b *testing.B) {
	db, err := gorm.Open(
		sqlite.Open("file:bench_scope"+time.Now().Format("150405.000")+"?mode=memory&cache=shared&loc=auto"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		b.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&TestBusinessData{}, &model.DataOwnership{}, &model.DataAccess{}, &model.Tenant{}); err != nil {
		b.Fatalf("migrate: %v", err)
	}

	// 种子：租户 1 拥有 100 条数据
	records := make([]TestBusinessData, 0, 100)
	owns := make([]model.DataOwnership, 0, 100)
	for i := 1; i <= 100; i++ {
		records = append(records, TestBusinessData{ID: uint64(i), Status: "active", Amount: i * 10})
		owns = append(owns, model.DataOwnership{
			DataID: uint64(i), DataType: "test_business_data", OwnerID: 1,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		})
	}
	if err := db.Create(&records).Error; err != nil {
		b.Fatalf("seed records: %v", err)
	}
	if err := db.Create(&owns).Error; err != nil {
		b.Fatalf("seed ownership: %v", err)
	}

	m := NewManager(db)
	ctx := WithTenant(context.Background(), 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var count int64
		if err := db.Model(&TestBusinessData{}).
			Scopes(m.TenantScope(ctx, "test_business_data")).
			Count(&count).Error; err != nil {
			b.Fatalf("scope query: %v", err)
		}
	}
}
