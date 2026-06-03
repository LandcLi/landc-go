package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/LandcLi/landc-go/frame/pkg/bootstrap"
	"github.com/LandcLi/landc-go/frame/pkg/trace"
)

// Command 命令行对象
type Command struct {
	Name          string               // 命令名称（区分大小写）
	Brief         string               // 命令简要描述
	Description   string               // 命令详细描述
	Func          Function             // 命令执行函数
	FuncWithValue FuncWithValue        // 带返回值的命令执行函数
	HelpFunc      Function             // 帮助信息函数
	Examples      string               // 使用示例
	Additional    string               // 附加信息
	Strict        bool                 // 严格模式，遇到无效选项时返回错误
	Config        string               // 配置节点名称
	EnableTrace   bool                 // 是否启用链路追踪
	Bootstrap     *bootstrap.Bootstrap // 自动初始化配置

	parent           *Command          // 父命令
	commands         []*Command        // 子命令
	arguments        []Argument        // 参数配置
	supportedOptions map[string]bool   // 支持的选项
	optionAliases    map[string]string // 选项别名
}

// Function 命令执行函数类型
type Function func(ctx context.Context, parser *Parser) error

// FuncWithValue 带返回值的命令执行函数类型
type FuncWithValue func(ctx context.Context, parser *Parser) (interface{}, error)

// Argument 参数配置
type Argument struct {
	Name        string   // 参数名称
	Brief       string   // 参数简要描述
	Default     string   // 默认值
	Optional    bool     // 是否可选
	IsValueList bool     // 是否为值列表
	Values      []string // 可选值列表
}

// NewCommand 创建一个新的命令对象
func NewCommand(name, brief string, f Function) *Command {
	return &Command{
		Name:             name,
		Brief:            brief,
		Func:             f,
		supportedOptions: make(map[string]bool),
		optionAliases:    make(map[string]string),
		Bootstrap:        bootstrap.New(),
	}
}

// AddCommand 添加子命令
func (c *Command) AddCommand(commands ...*Command) error {
	for _, cmd := range commands {
		cmd.parent = c
		c.commands = append(c.commands, cmd)
	}
	return nil
}

// AddArgument 添加参数配置
func (c *Command) AddArgument(arguments ...Argument) {
	c.arguments = append(c.arguments, arguments...)
}

// AddOption 添加选项配置
func (c *Command) AddOption(option string, hasValue bool) {
	c.supportedOptions[option] = hasValue
}

// Run 运行命令
func (c *Command) Run(ctx context.Context) error {
	return c.RunWithArgs(ctx, os.Args)
}

func (c *Command) RunWithBootstrap(ctx context.Context) error {
	if err := c.Bootstrap.Init(ctx); err != nil {
		return err
	}
	defer c.Bootstrap.Close()

	return c.RunWithArgs(ctx, os.Args)
}

func (c *Command) RunWithBootstrapArgs(ctx context.Context, args []string) error {
	if err := c.Bootstrap.Init(ctx); err != nil {
		return err
	}
	defer c.Bootstrap.Close()

	return c.RunWithArgs(ctx, args)
}

// RunWithArgs 使用指定的参数运行命令
func (c *Command) RunWithArgs(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("no command provided")
	}

	// 自动从环境变量恢复链路追踪上下文（如果有）
	if c.EnableTrace {
		ctx = InitTraceFromEnv(ctx)
		// 如果没有从环境变量恢复到上下文，则初始化新的追踪
		if trace.TraceID(ctx) == "" {
			ctx = trace.InitTrace(ctx)
		}
	}

	// 处理子命令
	if len(c.commands) > 0 && len(args) > 1 {
		subCmdName := args[1]
		for _, subCmd := range c.commands {
			if subCmd.Name == subCmdName {
				// 为子命令创建新的跨度
				if c.EnableTrace {
					ctx2, span := trace.NewSpan(ctx, subCmd.Name)
					ctx = ctx2
					defer span.End()
				}

				// 移除第一个参数（当前命令名）
				newArgs := append([]string{subCmd.Name}, args[2:]...)
				return subCmd.RunWithArgs(ctx, newArgs)
			}
		}
	}

	// 解析参数
	parserOption := ParserOption{
		Strict: c.Strict,
	}

	parser, err := ParseWithArgs(args, c.supportedOptions, parserOption)
	if err != nil {
		return err
	}

	// 执行命令
	if c.Func != nil {
		// 为当前命令创建跨度
		if c.EnableTrace {
			ctx, span := trace.NewSpan(ctx, c.Name)
			defer span.End()

			err := c.Func(ctx, parser)
			if err != nil {
				span.EndWithError(err)
			}
			return err
		}

		return c.Func(ctx, parser)
	}

	// 没有执行函数，显示帮助信息
	c.showHelp()
	return nil
}

// showHelp 显示帮助信息
func (c *Command) showHelp() {
	fmt.Printf("USAGE\n")
	fmt.Printf("    %s [OPTION]\n", c.Name)
	if c.Brief != "" {
		fmt.Printf("\nDESCRIPTION\n")
		fmt.Printf("    %s\n", c.Brief)
	}
	if c.Description != "" {
		fmt.Printf("\nDETAIL\n")
		fmt.Printf("    %s\n", c.Description)
	}
	if len(c.arguments) > 0 {
		fmt.Printf("\nARGUMENTS\n")
		for _, arg := range c.arguments {
			fmt.Printf("    %s\t%s\n", arg.Name, arg.Brief)
		}
	}
	if len(c.commands) > 0 {
		fmt.Printf("\nCOMMANDS\n")
		for _, cmd := range c.commands {
			fmt.Printf("    %s\t%s\n", cmd.Name, cmd.Brief)
		}
	}
	if c.Examples != "" {
		fmt.Printf("\nEXAMPLES\n")
		fmt.Printf("%s\n", c.Examples)
	}
	if c.Additional != "" {
		fmt.Printf("\nADDITIONAL\n")
		fmt.Printf("%s\n", c.Additional)
	}
}

// NewApp 创建一个新的应用命令
func NewApp() *Command {
	return &Command{
		Name:             os.Args[0],
		supportedOptions: make(map[string]bool),
		optionAliases:    make(map[string]string),
	}
}
