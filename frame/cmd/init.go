package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
)

type InitInput struct {
	Name     string `name:"name" brief:"Project name"`
	Path     string `name:"path" brief:"Project path (default: current directory)" optional:"true"`
	Module   string `name:"module" brief:"Go module name (default: project name)" optional:"true"`
	Force    bool   `name:"force" short:"f" brief:"Force overwrite existing files"`
	NoGit    bool   `name:"no-git" brief:"Skip git initialization"`
	NoReadme bool   `name:"no-readme" brief:"Skip README.md creation"`
}

func NewInitCommand() *cmd.Command {
	return cmd.NewCommand("init", "Initialize a new landc-go project", func(ctx context.Context, parser *cmd.Parser) error {
		input := &InitInput{
			Name: parser.GetArg(0),
		}

		if parser.HasOpt("path") {
			input.Path = parser.GetOpt("path")
		}
		if parser.HasOpt("module") {
			input.Module = parser.GetOpt("module")
		}
		if parser.HasOpt("force") || parser.HasOpt("f") {
			input.Force = true
		}
		if parser.HasOpt("no-git") {
			input.NoGit = true
		}
		if parser.HasOpt("no-readme") {
			input.NoReadme = true
		}

		return RunInit(ctx, input)
	})
}

func RunInit(ctx context.Context, input *InitInput) error {
	if input.Name == "" {
		return fmt.Errorf("project name is required")
	}

	projectPath := input.Path
	if projectPath == "" {
		projectPath = input.Name
	}

	moduleName := input.Module
	if moduleName == "" {
		moduleName = input.Name
	}

	fmt.Printf("Initializing landc-go project...\n")
	fmt.Printf("Project name: %s\n", input.Name)
	fmt.Printf("Project path: %s\n", projectPath)
	fmt.Printf("Module name: %s\n", moduleName)

	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if err := createProjectStructure(absPath, input.Name, moduleName, input); err != nil {
		return fmt.Errorf("failed to create project structure: %w", err)
	}

	if !input.NoGit {
		if err := initGit(absPath); err != nil {
			fmt.Printf("Warning: failed to initialize git: %v\n", err)
		}
	}

	// Run go mod tidy to generate go.sum
	if err := runGoModTidy(absPath); err != nil {
		fmt.Printf("Warning: go mod tidy failed: %v\n", err)
	}

	fmt.Printf("\n✓ Project initialized successfully!\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", projectPath)
	fmt.Printf("  go run main.go\n")

	return nil
}

func createProjectStructure(projectPath, projectName, moduleName string, input *InitInput) error {
	dirs := []string{
		"api/hello/v1",
		"service",
		"dao",
		"model",
		"sqls",
		"internal/cmd",
		"internal/controller/hello",
		"internal/service_impl/hello",
		"internal/dao_impl/hello",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(projectPath, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	createMap := []struct {
		name string
		fn   func(string, string) error
	}{
		{"main.go", func(p, m string) error { return createMainGo(p, moduleName) }},
		{"internal/cmd/cmd.go", func(p, m string) error { return createCmdFile(p, moduleName) }},
		{"go.mod", func(p, m string) error { return createGoMod(p, moduleName) }},
		{".gitignore", func(p, _ string) error { return os.WriteFile(filepath.Join(p, ".gitignore"), []byte(gitignoreContent), 0644) }},
		{"config.yaml", func(p, _ string) error { return os.WriteFile(filepath.Join(p, "config.yaml"), []byte(configYamlContent), 0644) }},
		{"sqls/init.sql", func(p, _ string) error { return os.WriteFile(filepath.Join(p, "sqls", "init.sql"), []byte(sqlInitContent), 0644) }},

		// routes are registered via meta.Meta tags, no routers.go needed
		{"api/hello/hello.go", func(p, m string) error { return createHelloApi(p, moduleName) }},
		{"api/hello/v1/say_hello.go", func(p, _ string) error { return os.WriteFile(filepath.Join(p, "api/hello/v1", "say_hello.go"), []byte(sayHelloContent), 0644) }},
		{"service/hello.go", func(p, m string) error { return createServiceInterface(p, moduleName) }},
		{"dao/hello.go", func(p, _ string) error { return createDaoInterface(p, moduleName) }},
		{"model/hello.go", func(p, _ string) error { return os.WriteFile(filepath.Join(p, "model/hello.go"), []byte(modelHelloContent), 0644) }},
		{"internal/impl.go", func(p, m string) error { return createInternalImpl(p, moduleName) }},
		{"internal/controller/hello/hello.go", func(p, m string) error { return createControllerImpl(p, moduleName) }},
		{"internal/service_impl/hello/hello.go", func(p, m string) error { return createServiceImpl(p, moduleName) }},
		{"internal/dao_impl/hello/hello.go", func(p, m string) error { return createDaoImpl(p, moduleName) }},
	}

	for _, cf := range createMap {
		if err := cf.fn(projectPath, moduleName); err != nil {
			return fmt.Errorf("failed to create %s: %w", cf.name, err)
		}
	}

	if !input.NoReadme {
		if err := createReadme(projectPath, projectName, moduleName); err != nil {
			return err
		}
	}

	return nil
}

// ==================== main.go ====================

func createMainGo(projectPath, moduleName string) error {
	content := fmt.Sprintf(`package main

import (
	"context"

	_ "%s/internal"
	"%s/internal/cmd"
)

func main() {
	cmd.Main.Run(context.Background())
}
`, moduleName, moduleName)
	return os.WriteFile(filepath.Join(projectPath, "main.go"), []byte(content), 0644)
}

func createCmdFile(projectPath, moduleName string) error {
	content := fmt.Sprintf(`package cmd

import (
	"context"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
	"github.com/LandcLi/landc-go/frame/pkg/web"
	"%s/api/hello"
)

// Main is the main application command.
// cmd.Main.Run(ctx) handles bootstrap lifecycle + signal handling.
var Main = cmd.NewCommand("main", "start HTTP server", func(ctx context.Context, parser *cmd.Parser) error {
	server := web.NewServer(nil)
	if err := server.RegisterHandler(hello.GetHelloController()); err != nil {
		return err
	}
	return server.RunWithContext(ctx)
})
`, moduleName)
	return os.WriteFile(filepath.Join(projectPath, "internal/cmd/cmd.go"), []byte(content), 0644)
}

// ==================== go.mod ====================

func createGoMod(projectPath, moduleName string) error {
	content := fmt.Sprintf(`module %s

go 1.24.0

require github.com/LandcLi/landc-go/frame v0.0.0
`, moduleName)
	return os.WriteFile(filepath.Join(projectPath, "go.mod"), []byte(content), 0644)
}

// landcGoDir returns the absolute path to the directory containing the CLI's go.mod.
// The CLI is at landc-go/frame/cmd/, so the frame module root is landc-go/frame/.

// ==================== static files ====================

const gitignoreContent = `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Go
vendor/

# Logs
logs/
`

const configYamlContent = `server:
  addr: "0.0.0.0"
  port: 8080

database:
  driver: "mysql"
  dsn: "root:password@tcp(localhost:3306)/database?charset=utf8mb4&parseTime=True&loc=Local"
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600
`

const sqlInitContent = `-- Example SQL for hello module
CREATE TABLE IF NOT EXISTS ` + "`hello`" + ` (
    ` + "`id`" + `         BIGINT       NOT NULL AUTO_INCREMENT,
    ` + "`name`" + `       VARCHAR(255) NOT NULL DEFAULT '',
    ` + "`created_at`" + ` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ` + "`updated_at`" + ` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (` + "`id`" + `)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`

const sayHelloContent = `package v1

import "github.com/LandcLi/landc-go/frame/pkg/meta"

type SayHelloRequest struct {
	meta.Meta ` + "`path:\"/api/hello/say\" method:\"POST\"`" + `
	Name      string ` + "`json:\"name\" binding:\"required\"`" + `
}

type SayHelloResponse struct {
	Message string ` + "`json:\"message\"`" + `
}
`

// ==================== api/routers.go ====================

// ==================== api/hello/hello.go ====================

func createHelloApi(projectPath, moduleName string) error {
	content := fmt.Sprintf(`package hello

import (
	"context"

	"github.com/LandcLi/landc-go/frame/pkg/di"
	v1 "%s/api/hello/v1"
)

// HelloController defines the hello service interface.
type HelloController interface {
	SayHello(ctx context.Context, req *v1.SayHelloRequest) (*v1.SayHelloResponse, error)
}

// HelloGateway is the service gateway for hello operations.
var HelloGateway = di.NewGateway[HelloController]("hello.controller")

// GetHelloController returns the registered HelloController implementation.
func GetHelloController() HelloController {
	return HelloGateway.Get()
}

// RegisterHelloController registers a HelloController implementation.
func RegisterHelloController(impl HelloController) {
	HelloGateway.Provide(impl)
}
`, moduleName)
	return os.WriteFile(filepath.Join(projectPath, "api/hello/hello.go"), []byte(content), 0644)
}

// ==================== service/hello.go ====================

func createServiceInterface(projectPath, moduleName string) error {
	content := fmt.Sprintf(`package service

import (
	"context"

	"github.com/LandcLi/landc-go/frame/pkg/di"
	v1 "%s/api/hello/v1"
)

// HelloService defines the hello business logic interface.
type HelloService interface {
	SayHello(ctx context.Context, req *v1.SayHelloRequest) (*v1.SayHelloResponse, error)
}

func GetHelloService() HelloService {
	return di.Require[HelloService]("hello.service")
}

func RegisterHelloService(s HelloService) {
	di.Provide[HelloService]("hello.service", s)
}
`, moduleName)
	return os.WriteFile(filepath.Join(projectPath, "service/hello.go"), []byte(content), 0644)
}

// ==================== dao/hello.go ====================

const daoHelloContent = `package dao

import (
	"context"

	"github.com/LandcLi/landc-go/frame/pkg/di"
)

// HelloDao defines the hello data access interface.
type HelloDao interface {
	SayHello(ctx context.Context, name string) (string, error)
}

func GetHelloDao() HelloDao {
	return di.Require[HelloDao]("hello.dao")
}

func RegisterHelloDao(impl HelloDao) {
	di.Provide[HelloDao]("hello.dao", impl)
}
`

func createDaoInterface(projectPath, _ string) error {
	return os.WriteFile(filepath.Join(projectPath, "dao/hello.go"), []byte(daoHelloContent), 0644)
}

// ==================== model/hello.go ====================

const modelHelloContent = `package model

import "time"

type Hello struct {
	ID        int64     ` + "`gorm:\"primaryKey;autoIncrement\" json:\"id\"`" + `
	Name      string    ` + "`gorm:\"size:255;not null\" json:\"name\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\"`" + `
}
`

// ==================== internal/impl.go ====================

func createInternalImpl(projectPath, moduleName string) error {
	content := fmt.Sprintf(`package internal

import (
	_ "%s/internal/controller/hello"
	_ "%s/internal/dao_impl/hello"
	_ "%s/internal/service_impl/hello"
)
`, moduleName, moduleName, moduleName)
	return os.WriteFile(filepath.Join(projectPath, "internal/impl.go"), []byte(content), 0644)
}

// ==================== internal/controller/hello/hello.go ====================

func createControllerImpl(projectPath, moduleName string) error {
	content := fmt.Sprintf(`package hello

import (
	"context"

	v1 "%s/api/hello/v1"
	helloApi "%s/api/hello"
	"%s/service"
)

type helloController struct{}

func init() {
	helloApi.RegisterHelloController(&helloController{})
}

func (c *helloController) SayHello(ctx context.Context, req *v1.SayHelloRequest) (*v1.SayHelloResponse, error) {
	return service.GetHelloService().SayHello(ctx, req)
}
`, moduleName, moduleName, moduleName)
	return os.WriteFile(filepath.Join(projectPath, "internal/controller/hello/hello.go"), []byte(content), 0644)
}

// ==================== internal/service_impl/hello/hello.go ====================

func createServiceImpl(projectPath, moduleName string) error {
	content := fmt.Sprintf(`package hello

import (
	"context"

	v1 "%s/api/hello/v1"
	"%s/dao"
	"%s/service"
)

type helloServiceImpl struct{}

func init() {
	service.RegisterHelloService(&helloServiceImpl{})
}

func (s *helloServiceImpl) SayHello(ctx context.Context, req *v1.SayHelloRequest) (*v1.SayHelloResponse, error) {
	msg, err := dao.GetHelloDao().SayHello(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	return &v1.SayHelloResponse{Message: msg}, nil
}
`, moduleName, moduleName, moduleName)
	return os.WriteFile(filepath.Join(projectPath, "internal/service_impl/hello/hello.go"), []byte(content), 0644)
}

// ==================== internal/dao_impl/hello/hello.go ====================

func createDaoImpl(projectPath, moduleName string) error {
	content := fmt.Sprintf(`package hello

import (
	"context"
	"fmt"

	"%s/dao"
)

type helloDaoImpl struct{}

func init() {
	dao.RegisterHelloDao(&helloDaoImpl{})
}

func (d *helloDaoImpl) SayHello(ctx context.Context, name string) (string, error) {
	return fmt.Sprintf("Hello, %%s!", name), nil
}
`, moduleName)
	return os.WriteFile(filepath.Join(projectPath, "internal/dao_impl/hello/hello.go"), []byte(content), 0644)
}

// ==================== README.md ====================

func createReadme(projectPath, projectName, moduleName string) error {
	content := fmt.Sprintf(`# %s

A landc-go project with layered architecture.

## Getting Started

### Prerequisites

- Go 1.24.0 or higher

### Installation

`+"```bash"+`
cd %s
go mod tidy
go run main.go
`+"```"+`

### Verify

`+"```bash"+`
# Health check
curl http://localhost:8080/health

# Say hello
curl -X POST http://localhost:8080/api/hello/say \
  -H "Content-Type: application/json" \
  -d '{"name":"World"}'
`+"```"+`

## Project Structure

`+"```"+`
.
├── main.go                    # Entry point: cmd.Main.Run(ctx)
├── config.yaml                # Server config
├── api/                       # Interface definitions
│   └── hello/
│       ├── hello.go           # HelloController interface + Gateway
│       └── v1/                # Request/Response types
│           └── say_hello.go
├── service/                   # Service interfaces
│   └── hello.go
├── dao/                       # DAO interfaces
│   └── hello.go
├── model/                     # Data models
│   └── hello.go
├── internal/                  # Internal implementations
│   ├── cmd/
│   │   └── cmd.go             # Main command definition
│   ├── impl.go                # Triggers init() of all subpackages
│   ├── controller/hello/      # Controller layer
│   ├── service_impl/hello/    # Service layer
│   └── dao_impl/hello/        # DAO layer
├── sqls/                      # SQL files
└── go.mod
`+"```"+`

## Architecture

### Call Chain

`+"```"+`
cmd.Main.Run(ctx)
  -> web.Server.RunWithBootstrap()
    -> Bootstrap Init
      -> config.InitGlobalConfigWithPath
      -> db.InitGlobalDBWithDefault
`+"```"+`

`+"```"+`
Request -> Gin Router -> Controller -> Service -> DAO -> Response
`+"```"+`

### Dependency Injection

All layers are registered via the DI container (github.com/LandcLi/landc-go/frame/pkg/di):

- Controller registers in internal/controller/hello/hello.go (init())
- Service registers in internal/service_impl/hello/hello.go (init())
- DAO registers in internal/dao_impl/hello/hello.go (init())

### Remote Mode

Generate SDK and switch to remote mode:

`+"```bash"+`
landc gen proxy --type=HelloController --gateway-name=hello.controller
`+"```"+`

In `+"`"+`internal/cmd/cmd.go`+"`"+`, replace local with remote:

`+"```go"+`
hello.HelloGateway.ProvideRemote("http://hello-service:8080")
ctrl := hello.HelloGateway.Get()  // same API as local
`+"```"+`

## License

MIT License
`, projectName, projectName)
	return os.WriteFile(filepath.Join(projectPath, "README.md"), []byte(content), 0644)
}

// ==================== go mod tidy ====================

func runGoModTidy(projectPath string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ==================== git init ====================

func initGit(projectPath string) error {
	gitDir := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return nil
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
