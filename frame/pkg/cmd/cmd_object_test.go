package cmd_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
)

// 测试结构体

type cMain struct {
	Meta struct {
		Name  string `name:"main"`
		Brief string `brief:"Main command"`
	} `name:"Meta"`
}

type cMainHttpInput struct {
	Meta struct {
		Name  string `name:"http"`
		Brief string `brief:"Start HTTP server"`
	} `name:"Meta"`
	Name string `name:"NAME" arg:"true" brief:"Server name"`
	Port int    `short:"p" name:"port" brief:"Port of HTTP server"`
}

type cMainHttpOutput struct{}

func (c *cMain) Http(ctx context.Context, in *cMainHttpInput) (*cMainHttpOutput, error) {
	fmt.Printf("Starting HTTP server %s on port %d\n", in.Name, in.Port)
	return &cMainHttpOutput{}, nil
}

type cMainGrpcInput struct {
	Meta struct {
		Name  string `name:"grpc"`
		Brief string `brief:"Start gRPC server"`
	} `name:"Meta"`
	Name string `name:"NAME" arg:"true" brief:"Server name"`
	Port int    `short:"p" name:"port" brief:"Port of gRPC server"`
}

type cMainGrpcOutput struct{}

func (c *cMain) Grpc(ctx context.Context, in *cMainGrpcInput) (*cMainGrpcOutput, error) {
	fmt.Printf("Starting gRPC server %s on port %d\n", in.Name, in.Port)
	return &cMainGrpcOutput{}, nil
}

// 测试带 Meta 字段的结构体

type cTest struct {
	Meta struct {
		Name  string `name:"test"`
		Brief string `brief:"Test command"`
	} `name:"Meta"`
}

type cTestEchoInput struct {
	Meta struct {
		Name  string `name:"echo"`
		Brief string `brief:"Echo message"`
	} `name:"Meta"`
	Message string `name:"MESSAGE" arg:"true" brief:"Message to echo"`
	Repeat  int    `short:"r" name:"repeat" brief:"Repeat count"`
}

type cTestEchoOutput struct{}

func (c *cTest) Echo(ctx context.Context, in *cTestEchoInput) (*cTestEchoOutput, error) {
	for i := 0; i < in.Repeat; i++ {
		fmt.Println(in.Message)
	}
	return &cTestEchoOutput{}, nil
}

func TestNewFromObject(t *testing.T) {
	// 保存原始参数
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// 创建对象
	mainObj := &cMain{}

	// 从对象创建命令
	cmd, err := cmd.NewFromObject(mainObj)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 检查命令名称
	if cmd.Name != "main" {
		t.Errorf("Expected command name 'main', got '%s'", cmd.Name)
	}

	// 检查命令简要描述
	if cmd.Brief != "Main command" {
		t.Errorf("Expected command brief 'Main command', got '%s'", cmd.Brief)
	}

	// 测试显示帮助信息，验证子命令是否被正确添加
	os.Args = []string{"main"}
	ctx := context.Background()
	err = cmd.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestCommandFromObjectRun(t *testing.T) {
	// 保存原始参数
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// 创建对象
	mainObj := &cMain{}

	// 从对象创建命令
	cmd, err := cmd.NewFromObject(mainObj)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 测试运行 http 命令
	os.Args = []string{"main", "http", "test-server", "--port=8080"}
	ctx := context.Background()
	err = cmd.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 测试运行 grpc 命令
	os.Args = []string{"main", "grpc", "test-server", "-p", "9090"}
	err = cmd.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestCommandFromObjectHelp(t *testing.T) {
	// 保存原始参数
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// 创建对象
	mainObj := &cMain{}

	// 从对象创建命令
	cmd, err := cmd.NewFromObject(mainObj)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 测试显示帮助信息
	os.Args = []string{"main"}
	ctx := context.Background()
	err = cmd.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 测试显示子命令帮助信息
	os.Args = []string{"main", "http"}
	err = cmd.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestCommandFromObjectWithMetaField(t *testing.T) {
	// 创建对象
	testObj := &cTest{}

	// 从对象创建命令
	cmd, err := cmd.NewFromObject(testObj)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 检查命令名称
	if cmd.Name != "test" {
		t.Errorf("Expected command name 'test', got '%s'", cmd.Name)
	}

	// 检查命令简要描述
	if cmd.Brief != "Test command" {
		t.Errorf("Expected command brief 'Test command', got '%s'", cmd.Brief)
	}

	// 测试运行 echo 命令
	os.Args = []string{"test", "echo", "Hello", "--repeat=3"}
	ctx := context.Background()
	err = cmd.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
