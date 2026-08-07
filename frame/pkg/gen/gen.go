package gen

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
)

// NewGenCommand 创建代码生成命令
// GenSubcommands 返回 gen 的全部子命令（api / service / dao / model / lib / all），
// 供 landc CLI 的 gen 命令挂载（避免 gen 命令嵌套 gen 导致 landc gen lib 不可用）。
func GenSubcommands() []*cmd.Command {
	return []*cmd.Command{
		newGenAPICommand(),
		newGenServiceCommand(),
		newGenDAOCommand(),
		newGenModelCommand(),
		newGenLibCommand(),
		newGenAllCommand(),
	}
}

// NewGenCommand 创建 gen 命令（含全部子命令）。
func NewGenCommand() *cmd.Command {
	genCmd := cmd.NewCommand("gen", "Generate code for api/service/dao/model layers", nil)

	if err := genCmd.AddCommand(
		newGenAPICommand(),
		newGenServiceCommand(),
		newGenDAOCommand(),
		newGenModelCommand(),
		newGenLibCommand(),
		newGenAllCommand(),
	); err != nil {
		return nil
	}

	return genCmd
}

func newGenAPICommand() *cmd.Command {
	command := cmd.NewCommand("api", "Generate API layer code", func(ctx context.Context, parser *cmd.Parser) error {
		name := parser.GetArg(0)
		if name == "" {
			return fmt.Errorf("model name is required, usage: landc gen api <name>")
		}
		module := getModuleOpt(parser)
		return generateAPI(name, module)
	})
	command.AddOption("module", true)
	return command
}

func newGenServiceCommand() *cmd.Command {
	command := cmd.NewCommand("service", "Generate Service layer code", func(ctx context.Context, parser *cmd.Parser) error {
		name := parser.GetArg(0)
		if name == "" {
			return fmt.Errorf("model name is required, usage: landc gen service <name>")
		}
		module := getModuleOpt(parser)
		return generateService(name, module)
	})
	command.AddOption("module", true)
	return command
}

func newGenDAOCommand() *cmd.Command {
	command := cmd.NewCommand("dao", "Generate DAO layer code", func(ctx context.Context, parser *cmd.Parser) error {
		name := parser.GetArg(0)
		if name == "" {
			return fmt.Errorf("model name is required, usage: landc gen dao <name>")
		}
		module := getModuleOpt(parser)
		return generateDAO(name, module)
	})
	command.AddOption("module", true)
	return command
}

func newGenModelCommand() *cmd.Command {
	command := cmd.NewCommand("model", "Generate model layer code", func(ctx context.Context, parser *cmd.Parser) error {
		name := parser.GetArg(0)
		if name == "" {
			return fmt.Errorf("model name is required, usage: landc gen model <name>")
		}
		module := getModuleOpt(parser)
		return generateModel(name, module)
	})
	command.AddOption("module", true)
	return command
}

func newGenLibCommand() *cmd.Command {
	command := cmd.NewCommand("lib", "Generate library-mode entry (serverlib/RegisterToRouter)", func(ctx context.Context, parser *cmd.Parser) error {
		name := parser.GetArg(0)
		if name == "" {
			return fmt.Errorf("service name is required, usage: landc gen lib <name>")
		}
		module := getModuleOpt(parser)
		return generateLib(name, module)
	})
	command.AddOption("module", true)
	return command
}

func newGenAllCommand() *cmd.Command {
	command := cmd.NewCommand("all", "Generate all layers (api + service + dao + model)", func(ctx context.Context, parser *cmd.Parser) error {
		name := parser.GetArg(0)
		if name == "" {
			return fmt.Errorf("model name is required, usage: landc gen all <name>")
		}
		module := getModuleOpt(parser)

		if err := generateModel(name, module); err != nil {
			return err
		}
		if err := generateDAO(name, module); err != nil {
			return err
		}
		if err := generateService(name, module); err != nil {
			return err
		}
		if err := generateAPI(name, module); err != nil {
			return err
		}

		fmt.Printf("\n✓ All layers generated for '%s'\n", name)

		// --check：生成后跑 go build 校验四层可编译
		if parser.HasOpt("check") {
			fmt.Println("  -> running `go build ./...` to verify generated code...")
			if err := runGoBuildCheck(); err != nil {
				return fmt.Errorf("generated code failed to build (see output above)")
			}
			fmt.Println("  -> build check passed")
		}
		return nil
	})
	command.AddOption("check", false)
	return command
}

// runGoBuildCheck 在当前目录执行 go build ./...，用于校验生成代码可编译。
func runGoBuildCheck() error {
	c := exec.Command("go", "build", "./...")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// ============ 模板数据 ============

type templateData struct {
	Name      string // 原始名称 e.g. "user"
	NameCamel string // 大驼峰 e.g. "User"
	NameLower string // 全小写 e.g. "user"
	NameSnake string // 蛇形 e.g. "user_order"
	Module    string // Go module 名
}

func newTemplateData(name, module string) *templateData {
	return &templateData{
		Name:      name,
		NameCamel: toCamelCase(name),
		NameLower: strings.ToLower(name),
		NameSnake: toSnakeCase(name),
		Module:    module,
	}
}

// ============ 生成器 ============

func generateAPI(name, module string) error {
	data := newTemplateData(name, module)

	// 创建目录
	apiDir := filepath.Join("api", data.NameLower, "v1")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		return fmt.Errorf("create api dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join("api", data.NameLower), 0o755); err != nil {
		return fmt.Errorf("create api subdir: %w", err)
	}

	// 生成 api/{name}/{name}.go
	if err := renderToFile(filepath.Join("api", data.NameLower, data.NameLower+".go"), tplAPIInterface, data); err != nil {
		return err
	}

	// 生成 api/{name}/v1/request.go
	if err := renderToFile(filepath.Join(apiDir, "request.go"), tplAPIRequest, data); err != nil {
		return err
	}

	// 生成 api/{name}/v1/response.go
	if err := renderToFile(filepath.Join(apiDir, "response.go"), tplAPIResponse, data); err != nil {
		return err
	}

	fmt.Printf("  ✓ API layer generated: api/%s/\n", data.NameLower)
	return nil
}

func generateService(name, module string) error {
	data := newTemplateData(name, module)

	if err := os.MkdirAll("service", 0o755); err != nil {
		return fmt.Errorf("create service dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join("internal", "service"), 0o755); err != nil {
		return fmt.Errorf("create internal/service dir: %w", err)
	}

	// 生成 service/{name}_service.go（接口）
	if err := renderToFile(filepath.Join("service", data.NameLower+"_service.go"), tplServiceInterface, data); err != nil {
		return err
	}

	// 生成 internal/service/{name}_service_impl.go（实现）
	if err := renderToFile(filepath.Join("internal", "service", data.NameLower+"_service_impl.go"), tplServiceImpl, data); err != nil {
		return err
	}

	fmt.Printf("  ✓ Service layer generated: service/%s_service.go\n", data.NameLower)
	return nil
}

func generateDAO(name, module string) error {
	data := newTemplateData(name, module)

	if err := os.MkdirAll("dao", 0o755); err != nil {
		return fmt.Errorf("create dao dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join("internal", "dao"), 0o755); err != nil {
		return fmt.Errorf("create internal/dao dir: %w", err)
	}

	// 生成 dao/{name}_dao.go（接口）
	if err := renderToFile(filepath.Join("dao", data.NameLower+"_dao.go"), tplDAOInterface, data); err != nil {
		return err
	}

	// 生成 internal/dao/{name}_dao_impl.go（实现）
	if err := renderToFile(filepath.Join("internal", "dao", data.NameLower+"_dao_impl.go"), tplDAOImpl, data); err != nil {
		return err
	}

	fmt.Printf("  ✓ DAO layer generated: dao/%s_dao.go\n", data.NameLower)
	return nil
}

func generateLib(name, module string) error {
	data := newTemplateData(name, module)

	if err := os.MkdirAll("serverlib", 0o755); err != nil {
		return fmt.Errorf("create serverlib dir: %w", err)
	}
	if err := renderToFile(filepath.Join("serverlib", "register.go"), tplLibrary, data); err != nil {
		return err
	}

	fmt.Printf("  ✓ Library entry generated: serverlib/register.go\n")
	fmt.Println("  -> 请补全：控制器清单（一个服务可能有多个 controller，逐一注册）、鉴权中间件与初始化")
	return nil
}

func generateModel(name, module string) error {
	data := newTemplateData(name, module)

	if err := os.MkdirAll("model", 0o755); err != nil {
		return fmt.Errorf("create model dir: %w", err)
	}

	if err := renderToFile(filepath.Join("model", data.NameLower+".go"), tplModel, data); err != nil {
		return err
	}

	fmt.Printf("  ✓ Model generated: model/%s.go\n", data.NameLower)
	return nil
}

// ============ 工具函数 ============

func renderToFile(path, tplContent string, data interface{}) error {
	// 如果文件已存在，不覆盖
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("  - Skipped (exists): %s\n", path)
		return nil
	}

	tmpl, err := template.New("").Parse(tplContent)
	if err != nil {
		return fmt.Errorf("parse template failed: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template failed: %w", err)
	}

	content := buf.Bytes()
	// 生成的 Go 代码自动 gofmt；异常时保留原文并提示，不中断生成
	if formatted, err := format.Source(content); err == nil {
		content = formatted
	} else {
		fmt.Printf("  - Warning: gofmt failed for %s (please format manually): %v\n", path, err)
	}

	//nolint:gosec // 生成文件无敏感信息，0644 便于团队共享
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write file failed: %w", err)
	}
	return nil
}

func detectModule() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "myproject"
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return "myproject"
}

func getModuleOpt(parser *cmd.Parser) string {
	if parser.HasOpt("module") {
		return parser.GetOpt("module")
	}
	return detectModule()
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func toSnakeCase(s string) string {
	var result []byte
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(c-'A'+'a'))
		} else {
			result = append(result, string(c)...)
		}
	}
	return string(result)
}
