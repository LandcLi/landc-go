package dao

import (
	"context"

	"github.com/LandcLi/landc-go/frame/pkg/di"

	"github.com/LandcLi/landc-go/examples/demo/model"
)

// HelloDao 定义 hello 数据访问接口
type HelloDao interface {
	GetByName(ctx context.Context, name string) (*model.Hello, error)
	Create(ctx context.Context, name string) (*model.Hello, error)
}

func GetHelloDao() HelloDao {
	return di.Require[HelloDao]("hello.dao")
}

func RegisterHelloDao(impl HelloDao) {
	di.Provide[HelloDao]("hello.dao", impl)
}
