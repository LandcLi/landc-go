package hello

import (
	"context"

	"github.com/LandcLi/landc-go/frame/pkg/db"

	"github.com/LandcLi/landc-go/examples/demo/dao"
	"github.com/LandcLi/landc-go/examples/demo/model"
)

type helloDaoImpl struct{}

func init() {
	dao.RegisterHelloDao(&helloDaoImpl{})
}

// GetByName 使用 db.GetTx(ctx)：有事务时用事务连接，否则普通连接。
// 无记录返回 gorm.ErrRecordNotFound。
func (d *helloDaoImpl) GetByName(ctx context.Context, name string) (*model.Hello, error) {
	var h model.Hello
	if err := db.GetTx(ctx).Where("name = ?", name).First(&h).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

func (d *helloDaoImpl) Create(ctx context.Context, name string) (*model.Hello, error) {
	h := &model.Hello{Name: name}
	if err := db.GetTx(ctx).Create(h).Error; err != nil {
		return nil, err
	}
	return h, nil
}
