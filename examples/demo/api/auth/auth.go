package auth

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/LandcLi/landc-go/frame/pkg/di"

	v1 "github.com/LandcLi/landc-go/examples/demo/api/auth/v1"
)

// AuthController 定义认证相关接口
// Profile 注入 *gin.Context：需要读取 middleware.Auth 写入的用户信息
type AuthController interface {
	Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error)
	Profile(g *gin.Context, req *v1.ProfileRequest) (*v1.ProfileResponse, error)
}

var AuthGateway = di.NewGateway[AuthController]("auth.controller")

func GetAuthController() AuthController {
	return AuthGateway.Get()
}

func RegisterAuthController(impl AuthController) {
	AuthGateway.Provide(impl)
}
