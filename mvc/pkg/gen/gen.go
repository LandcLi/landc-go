package gen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/LandcLi/landc-go/mvc/pkg/cmd"
)

// NewGenCommand 创建代码生成命令
func NewGenCommand() *cmd.Command {
	genCmd := cmd.NewCommand("gen", "Generate code for api/service/dao layers", nil)

	genCmd.AddCommand(
		newGenAPICommand(),
		newGenServiceCommand(),
		newGenDAOCommand(),
		newGenAllCommand(),
	)

	return genCmd
}

func newGenAPICommand() *cmd.Command {
	return cmd.NewCommand("api", "Generate API layer code", func(ctx context.Context, parser *cmd.Parser) error {
		name := parser.GetArg(0)
		if name == "" {
			return fmt.Errorf("model name is required, usage: landc gen api <name>")
		}
		module := getModuleOpt(parser)
		return generateAPI(name, module)
	})
}

func newGenServiceCommand() *cmd.Command {
	return cmd.NewCommand("service", "Generate Service layer code", func(ctx context.Context, parser *cmd.Parser) error {
		name := parser.GetArg(0)
		if name == "" {
			return fmt.Errorf("model name is required, usage: landc gen service <name>")
		}
		module := getModuleOpt(parser)
		return generateService(name, module)
	})
}

func newGenDAOCommand() *cmd.Command {
	return cmd.NewCommand("dao", "Generate DAO layer code", func(ctx context.Context, parser *cmd.Parser) error {
		name := parser.GetArg(0)
		if name == "" {
			return fmt.Errorf("model name is required, usage: landc gen dao <name>")
		}
		module := getModuleOpt(parser)
		return generateDAO(name, module)
	})
}

func newGenAllCommand() *cmd.Command {
	return cmd.NewCommand("all", "Generate all layers (api + service + dao + model)", func(ctx context.Context, parser *cmd.Parser) error {
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
		return nil
	})
}

// ============ 模板数据 ============

type templateData struct {
	Name       string // 原始名称 e.g. "user"
	NameCamel  string // 大驼峰 e.g. "User"
	NameLower  string // 全小写 e.g. "user"
	NameSnake  string // 蛇形 e.g. "user_order"
	Module     string // Go module 名
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
	os.MkdirAll(apiDir, 0755)
	os.MkdirAll(filepath.Join("api", data.NameLower), 0755)

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

	os.MkdirAll("service", 0755)
	os.MkdirAll(filepath.Join("internal", "service"), 0755)

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

	os.MkdirAll("dao", 0755)
	os.MkdirAll(filepath.Join("internal", "dao"), 0755)

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

func generateModel(name, module string) error {
	data := newTemplateData(name, module)

	os.MkdirAll("model", 0755)

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

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file failed: %w", err)
	}
	defer f.Close()

	return tmpl.Execute(f, data)
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
		if len(parts[i]) > 0 {
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
			result = append(result, byte(c+32))
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}
