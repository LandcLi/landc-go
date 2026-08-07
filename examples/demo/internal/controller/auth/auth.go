package auth

import (
	"context"

	"github.com/gin-gonic/gin"

	authApi "github.com/LandcLi/landc-go/examples/demo/api/auth"
	v1 "github.com/LandcLi/landc-go/examples/demo/api/auth/v1"
	"github.com/LandcLi/landc-go/examples/demo/service"
)

type authController struct{}

func init() {
	authApi.RegisterAuthController(&authController{})
}

func (c *authController) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	token, err := service.GetAuthService().Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	return &v1.LoginResponse{Token: token}, nil
}

// Profile 注入 *gin.Context：middleware.Auth 已把用户信息写入 gin.Context
func (c *authController) Profile(g *gin.Context, req *v1.ProfileRequest) (*v1.ProfileResponse, error) {
	return &v1.ProfileResponse{
		UserID:   g.GetUint("user_id"),
		Username: g.GetString("username"),
		Role:     g.GetString("role"),
	}, nil
}
