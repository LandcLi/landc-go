package hello

import (
	"context"
	"fmt"

	v1 "github.com/LandcLi/landc-go/examples/demo/api/hello/v1"
	helloApi "github.com/LandcLi/landc-go/examples/demo/api/hello"
	"github.com/LandcLi/landc-go/examples/demo/service"
)

type helloController struct{}

func init() {
	helloApi.RegisterHelloController(&helloController{})
}

func (c *helloController) SayHello(ctx context.Context, req *v1.SayHelloRequest) (*v1.SayHelloResponse, error) {
	h, hit, err := service.GetHelloService().SayHello(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	return &v1.SayHelloResponse{
		Message: fmt.Sprintf("Hello, %s! (id=%d)", h.Name, h.ID),
		ID:      h.ID,
		Hit:     hit,
	}, nil
}
