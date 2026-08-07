package hello

import (
	"context"
	"errors"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/cache"
	"gorm.io/gorm"

	"github.com/LandcLi/landc-go/examples/demo/dao"
	"github.com/LandcLi/landc-go/examples/demo/model"
	"github.com/LandcLi/landc-go/examples/demo/service"
)

type helloServiceImpl struct{}

func init() {
	service.RegisterHelloService(&helloServiceImpl{})
}

// SayHello 先查缓存，未命中查库；不存在则创建（首次呼叫入库）。
// 返回 (数据, 是否缓存命中, error)，演示 db.GetTx + cache.GetCacheFrom 的 ctx 资源解析。
func (s *helloServiceImpl) SayHello(ctx context.Context, name string) (*model.Hello, bool, error) {
	key := "hello:" + name

	// 1. 缓存优先
	if c := cache.GetCacheFrom(ctx); c != nil {
		var cached model.Hello
		if err := c.GetObject(ctx, key, &cached); err == nil {
			return &cached, true, nil
		}
	}

	// 2. 查库；不存在则创建
	h, err := dao.GetHelloDao().GetByName(ctx, name)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		h, err = dao.GetHelloDao().Create(ctx, name)
	}
	if err != nil {
		return nil, false, err
	}

	// 3. 回写缓存（10 分钟）
	if c := cache.GetCacheFrom(ctx); c != nil {
		_ = c.SetObject(ctx, key, h, 10*time.Minute)
	}
	return h, false, nil
}
