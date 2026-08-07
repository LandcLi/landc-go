package service

import (
	"context"

	"github.com/LandcLi/landc-go/frame/pkg/di"
)

// AuthService 定义认证业务接口
type AuthService interface {
	Login(ctx context.Context, username, password string) (string, error)
}

func GetAuthService() AuthService {
	return di.Require[AuthService]("auth.service")
}

func RegisterAuthService(s AuthService) {
	di.Provide[AuthService]("auth.service", s)
}
