package service

import (
	"context"

	"github.com/LandcLi/landc-go/frame/pkg/di"

	"github.com/LandcLi/landc-go/examples/demo/model"
)

// HelloService 定义 hello 业务接口
type HelloService interface {
	SayHello(ctx context.Context, name string) (*model.Hello, bool, error)
}

func GetHelloService() HelloService {
	return di.Require[HelloService]("hello.service")
}

func RegisterHelloService(s HelloService) {
	di.Provide[HelloService]("hello.service", s)
}
