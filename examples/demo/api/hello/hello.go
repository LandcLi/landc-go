package hello

import (
	"context"

	"github.com/LandcLi/landc-go/frame/pkg/di"

	v1 "github.com/LandcLi/landc-go/examples/demo/api/hello/v1"
)

// HelloController 定义 hello 服务接口
type HelloController interface {
	SayHello(ctx context.Context, req *v1.SayHelloRequest) (*v1.SayHelloResponse, error)
}

// HelloGateway 是 hello 服务的 DI 网关
var HelloGateway = di.NewGateway[HelloController]("hello.controller")

func GetHelloController() HelloController {
	return HelloGateway.Get()
}

func RegisterHelloController(impl HelloController) {
	HelloGateway.Provide(impl)
}
