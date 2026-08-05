package gen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAll(t *testing.T) {
	// 在临时目录中测试代码生成
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// 创建 go.mod
	_ = os.WriteFile("go.mod", []byte("module testproject\n\ngo 1.24.0\n"), 0o600)

	// 生成所有层
	if err := generateModel("order", "testproject"); err != nil {
		t.Fatalf("generateModel failed: %v", err)
	}
	if err := generateDAO("order", "testproject"); err != nil {
		t.Fatalf("generateDAO failed: %v", err)
	}
	if err := generateService("order", "testproject"); err != nil {
		t.Fatalf("generateService failed: %v", err)
	}
	if err := generateAPI("order", "testproject"); err != nil {
		t.Fatalf("generateAPI failed: %v", err)
	}

	// 验证文件存在
	expectedFiles := []string{
		"model/order.go",
		"dao/order_dao.go",
		"internal/dao/order_dao_impl.go",
		"service/order_service.go",
		"internal/service/order_service_impl.go",
		"api/order/order.go",
		"api/order/v1/request.go",
		"api/order/v1/response.go",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file not generated: %s", f)
		}
	}
}

func TestGenerateLib(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	_ = os.WriteFile("go.mod", []byte("module testproject\n\ngo 1.24.0\n"), 0o600)

	if err := generateLib("user", "testproject"); err != nil {
		t.Fatalf("generateLib failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "serverlib", "register.go"))
	if err != nil {
		t.Fatalf("read serverlib/register.go: %v", err)
	}
	s := string(content)
	for _, want := range []string{
		"func RegisterToRouter",
		"func WithAuth",
		"func WithWebOptions",
		"user.GetUserController()",
		`_ "testproject/internal"`,
		"frameconfig.GetConfig()",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("generated serverlib missing %q", want)
		}
	}
}

func TestGenerateSkipsExisting(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	_ = os.WriteFile("go.mod", []byte("module testproject\n\ngo 1.24.0\n"), 0o600)

	// 第一次生成
	generateModel("product", "testproject")

	// 读取内容
	content1, _ := os.ReadFile(filepath.Join(tmpDir, "model", "product.go"))

	// 第二次生成（应跳过已存在文件）
	generateModel("product", "testproject")

	content2, _ := os.ReadFile(filepath.Join(tmpDir, "model", "product.go"))

	if !bytes.Equal(content1, content2) {
		t.Error("Existing file should not be overwritten")
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user", "User"},
		{"user_order", "UserOrder"},
		{"hello_world", "HelloWorld"},
		{"a", "A"},
	}

	for _, tt := range tests {
		result := toCamelCase(tt.input)
		if result != tt.expected {
			t.Errorf("toCamelCase(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user", "user"},
		{"UserOrder", "user_order"},
		{"helloWorld", "hello_world"},
	}

	for _, tt := range tests {
		result := toSnakeCase(tt.input)
		if result != tt.expected {
			t.Errorf("toSnakeCase(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestDetectModule(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	_ = os.WriteFile("go.mod", []byte("module github.com/example/myapp\n\ngo 1.24.0\n"), 0o600)

	result := detectModule()
	if result != "github.com/example/myapp" {
		t.Errorf("detectModule() = %s, want github.com/example/myapp", result)
	}
}
