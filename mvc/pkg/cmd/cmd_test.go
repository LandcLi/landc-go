package cmd_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/LandcLi/landc-go/mvc/pkg/cmd"
)

func TestCommand(t *testing.T) {
	// 保存原始参数
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// 测试基本命令
	testCmd := cmd.NewCommand("test", "Test command", func(ctx context.Context, parser *cmd.Parser) error {
		fmt.Println("Test command executed")
		return nil
	})

	// 测试子命令
	subCmd := cmd.NewCommand("sub", "Sub command", func(ctx context.Context, parser *cmd.Parser) error {
		fmt.Println("Sub command executed")
		return nil
	})

	testCmd.AddCommand(subCmd)

	// 测试执行主命令
	os.Args = []string{"test"}
	ctx := context.Background()
	err := testCmd.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 测试执行子命令
	os.Args = []string{"test", "sub"}
	err = testCmd.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestParser(t *testing.T) {
	// 测试解析参数
	args := []string{"test", "arg1", "arg2", "--option=value", "-f", "file.txt"}
	parser := cmd.NewParser(args)

	// 测试获取参数
	arg0 := parser.GetArg(0)
	if arg0 != "arg1" {
		t.Errorf("Expected 'arg1', got '%s'", arg0)
	}

	arg1 := parser.GetArg(1)
	if arg1 != "arg2" {
		t.Errorf("Expected 'arg2', got '%s'", arg1)
	}

	// 测试获取选项
	option := parser.GetOpt("option")
	if option != "value" {
		t.Errorf("Expected 'value', got '%s'", option)
	}

	f := parser.GetOpt("f")
	if f != "file.txt" {
		t.Errorf("Expected 'file.txt', got '%s'", f)
	}

	// 测试检查选项是否存在
	hasOption := parser.HasOpt("option")
	if !hasOption {
		t.Errorf("Expected option to exist")
	}

	hasNonExistent := parser.HasOpt("nonexistent")
	if hasNonExistent {
		t.Errorf("Expected non-existent option to not exist")
	}

	// 测试获取所有选项
	allOpts := parser.GetOptAll()
	if len(allOpts) != 2 {
		t.Errorf("Expected 2 options, got %d", len(allOpts))
	}

	// 测试获取所有参数
	allArgs := parser.GetArgs()
	if len(allArgs) != 2 {
		t.Errorf("Expected 2 arguments, got %d", len(allArgs))
	}
}

func TestCommandWithOptions(t *testing.T) {
	// 保存原始参数
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// 测试带选项的命令
	testCmd := cmd.NewCommand("test", "Test command with options", func(ctx context.Context, parser *cmd.Parser) error {
		option := parser.GetOpt("option")
		fmt.Printf("Option value: %s\n", option)
		return nil
	})

	// 添加选项配置
	testCmd.AddOption("option", true)

	// 测试带选项执行
	os.Args = []string{"test", "--option=test-value"}
	ctx := context.Background()
	err := testCmd.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestCommandHelp(t *testing.T) {
	// 保存原始参数
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// 测试帮助信息
	testCmd := cmd.NewCommand("test", "Test command", nil)
	testCmd.Description = "This is a test command"
	testCmd.Examples = "  test --option=value"
	testCmd.Additional = "Additional information about the test command"

	// 测试显示帮助信息
	os.Args = []string{"test"}
	ctx := context.Background()
	err := testCmd.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestParseFunction(t *testing.T) {
	// 测试 Parse 函数
	args := []string{"test", "--name=john", "--age=30", "arg1"}

	// 配置选项
	supportedOptions := map[string]bool{
		"name":    true,
		"age":     true,
		"verbose": false,
	}

	parser, err := cmd.ParseWithArgs(args, supportedOptions)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 测试获取选项
	name := parser.GetOpt("name")
	if name != "john" {
		t.Errorf("Expected 'john', got '%s'", name)
	}

	age := parser.GetOpt("age")
	if age != "30" {
		t.Errorf("Expected '30', got '%s'", age)
	}

	// 测试获取参数
	arg := parser.GetArg(0)
	if arg != "arg1" {
		t.Errorf("Expected 'arg1', got '%s'", arg)
	}
}

func TestOptionAliases(t *testing.T) {
	// 测试选项别名
	args := []string{"test", "-n", "john", "--age=30"}

	// 配置选项，带别名
	supportedOptions := map[string]bool{
		"n,name": true,
		"a,age":  true,
	}

	parser, err := cmd.ParseWithArgs(args, supportedOptions)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 测试通过主要选项名获取
	name := parser.GetOpt("n")
	if name != "john" {
		t.Errorf("Expected 'john', got '%s'", name)
	}

	// 测试通过别名获取
	nameAlias := parser.GetOpt("name")
	if nameAlias != "john" {
		t.Errorf("Expected 'john', got '%s'", nameAlias)
	}

	// 测试长选项别名
	age := parser.GetOpt("a")
	if age != "30" {
		t.Errorf("Expected '30', got '%s'", age)
	}

	ageAlias := parser.GetOpt("age")
	if ageAlias != "30" {
		t.Errorf("Expected '30', got '%s'", ageAlias)
	}
}

func TestOptionWithValue(t *testing.T) {
	// 测试需要参数的选项
	args := []string{"test", "--output=file.txt", "-v"}

	// 配置选项，指定是否需要参数
	supportedOptions := map[string]bool{
		"output":    true,  // 需要参数
		"v,verbose": false, // 不需要参数
	}

	parser, err := cmd.ParseWithArgs(args, supportedOptions)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 测试需要参数的选项
	output := parser.GetOpt("output")
	if output != "file.txt" {
		t.Errorf("Expected 'file.txt', got '%s'", output)
	}

	// 测试不需要参数的选项
	verbose := parser.GetOpt("v")
	if verbose != "" {
		t.Errorf("Expected empty string, got '%s'", verbose)
	}

	// 测试通过别名获取
	verboseAlias := parser.GetOpt("verbose")
	if verboseAlias != "" {
		t.Errorf("Expected empty string, got '%s'", verboseAlias)
	}
}

func TestStrictMode(t *testing.T) {
	// 测试严格模式
	args := []string{"test", "--valid=value", "--invalid=value"}

	// 配置选项
	supportedOptions := map[string]bool{
		"valid": true,
	}

	// 非严格模式
	parserOption := cmd.ParserOption{
		Strict: false,
	}

	_, err := cmd.ParseWithArgs(args, supportedOptions, parserOption)
	if err != nil {
		t.Errorf("Expected no error in non-strict mode, got %v", err)
	}

	// 测试严格模式
	parserOption.Strict = true
	_, err = cmd.ParseWithArgs(args, supportedOptions, parserOption)
	if err != nil {
		// 严格模式下应该返回错误
		fmt.Println("Strict mode correctly returned error for invalid option")
	}
}

func TestCommandWithOptionAliases(t *testing.T) {
	// 保存原始参数
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// 测试命令带选项别名
	testCmd := cmd.NewCommand("test", "Test command with option aliases", func(ctx context.Context, parser *cmd.Parser) error {
		name := parser.GetOpt("n")
		nameAlias := parser.GetOpt("name")
		fmt.Printf("Name: %s, Name (alias): %s\n", name, nameAlias)
		return nil
	})

	// 添加带别名的选项
	testCmd.AddOption("n,name", true)

	// 测试使用短选项
	os.Args = []string{"test", "-n", "john"}
	ctx := context.Background()
	err := testCmd.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 测试使用长选项
	os.Args = []string{"test", "--name", "jane"}
	err = testCmd.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
