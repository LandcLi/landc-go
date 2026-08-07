package auth

import (
	"context"
	"errors"

	"github.com/LandcLi/landc-go/frame/pkg/auth"

	"github.com/LandcLi/landc-go/examples/demo/service"
)

type authServiceImpl struct{}

func init() {
	service.RegisterAuthService(&authServiceImpl{})
}

// demoUsers 演示用静态账号；真实项目请改为数据库校验 + 密码哈希。
var demoUsers = map[string]string{
	"admin": "admin123",
	"landc": "landc123",
}

func (s *authServiceImpl) Login(ctx context.Context, username, password string) (string, error) {
	pwd, ok := demoUsers[username]
	if !ok || pwd != password {
		return "", errors.New("invalid username or password")
	}
	// 签发 JWT：JWT 配置由 bootstrap 从 config.yaml 的 jwt 段自动初始化
	token, err := auth.GenerateToken(10001, username, "admin")
	if err != nil {
		return "", err
	}
	return token, nil
}
