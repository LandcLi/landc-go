package proxygen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateIncludesExplicitConstructor(t *testing.T) {
	tmpDir := t.TempDir()

	// 模拟项目结构：go.mod + api/user 接口包 + api/user/v1 请求响应包
	apiDir := filepath.Join(tmpDir, "api", "user")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	goMod := "module example.com/demo\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	iface := `package user

import (
	"context"

	"example.com/demo/api/user/v1"
)

// UserController 用户接口
type UserController interface {
	Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error)
}
`
	if err := os.WriteFile(filepath.Join(apiDir, "controller.go"), []byte(iface), 0o644); err != nil {
		t.Fatal(err)
	}

	v1Dir := filepath.Join(apiDir, "v1")
	if err := os.MkdirAll(v1Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	v1File := `package v1

// LoginRequest 登录请求
type LoginRequest struct {
	Username string ` + "`json:\"username\"`" + `
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string ` + "`json:\"token\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(v1Dir, "request.go"), []byte(v1File), 0o644); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(tmpDir, "sdk", "user_proxy_gen.go")
	err := Generate(Config{
		InterfaceName: "UserController",
		GatewayName:   "user.controller",
		Dir:           apiDir,
		Output:        output,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	code, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	s := string(code)

	// 需求 5：显式构造函数
	if !strings.Contains(s, "func NewUserControllerProxy(baseURL string, opts ...di.RemoteOption) user.UserController {") {
		t.Error("generated code missing NewUserControllerProxy explicit constructor")
	}
	if !strings.Contains(s, "di.NewProxyClient(baseURL, opts...)") {
		t.Error("generated constructor should use di.NewProxyClient")
	}
	// 向后兼容：init 注册保留
	if !strings.Contains(s, "func init()") {
		t.Error("generated code missing init() factory registration")
	}
	if !strings.Contains(s, "di.RegisterProxyFactory") {
		t.Error("generated code missing RegisterProxyFactory call")
	}
}
