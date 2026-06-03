package main

import (
	"context"
	"fmt"
	"os"
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

	fmt.Printf("\n✓ Project initialized successfully!\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", projectPath)
	fmt.Printf("  go mod tidy\n")
	fmt.Printf("  go run main.go\n")

	return nil
}

func createProjectStructure(projectPath, projectName, moduleName string, input *InitInput) error {
	dirs := []string{
		"api/hello/v1",
		"service",
		"dao",
		"model",
		"utility/config",
		"utility/db",
		"utility/logger",
		"utility/cache",
		"utility/validator",
		"utility/response",
		"utility/error",
		"utility/auth",
		"utility/utils",
		"internal/controller",
		"internal/service",
		"internal/dao",
		"test",
		"logs",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(projectPath, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	if err := createMainGo(projectPath, projectName, moduleName); err != nil {
		return err
	}

	if err := createGoMod(projectPath, moduleName); err != nil {
		return err
	}

	if err := createConfigYaml(projectPath); err != nil {
		return err
	}

	if err := createApiFiles(projectPath, moduleName); err != nil {
		return err
	}

	if err := createInternalFiles(projectPath, moduleName); err != nil {
		return err
	}

	if !input.NoReadme {
		if err := createReadme(projectPath, projectName); err != nil {
			return err
		}
	}

	return nil
}

func createMainGo(projectPath, projectName, moduleName string) error {
	content := fmt.Sprintf(`package main

import (
	"context"
	"fmt"

	"%s/api"
	"%s/internal"

	"github.com/LandcLi/landc-go/frame/pkg/web"
)

func main() {
	ctx := context.Background()

	server := web.NewServer(nil)
	
	server.Bootstrap().
		BeforeInit(func(ctx context.Context) error {
			fmt.Println("Initializing application...")
			return nil
		}).
		AfterInit(func(ctx context.Context) error {
			fmt.Println("Application initialized successfully")
			return nil
		}).
		BeforeRun(func(ctx context.Context) error {
			fmt.Println("Server starting...")
			return nil
		}).
		AfterRun(func(ctx context.Context) error {
			fmt.Println("Server stopped")
			return nil
		})

	api.RegisterDefaultRouters(server)

	if err := server.RunWithBootstrap(); err != nil {
		panic(err)
	}
}
`, moduleName, moduleName)

	return os.WriteFile(filepath.Join(projectPath, "main.go"), []byte(content), 0644)
}

func createBootstrapExample(projectPath, moduleName string) error {
	content := fmt.Sprintf(`package main

import (
	"context"
	"fmt"

	"%s/api"
	"%s/internal"

	"github.com/LandcLi/landc-go/frame/pkg/web"
)

func main() {
	ctx := context.Background()

	server := web.NewServer(nil)
	
	server.Bootstrap().
		BeforeInit(func(ctx context.Context) error {
			fmt.Println("Before init hook")
			return nil
		}).
		AfterInit(func(ctx context.Context) error {
			fmt.Println("After init hook")
			return nil
		}).
		BeforeRun(func(ctx context.Context) error {
			fmt.Println("Before run hook")
			return nil
		}).
		AfterRun(func(ctx context.Context) error {
			fmt.Println("After run hook")
			return nil
		})

	api.RegisterDefaultRouters(server)

	if err := server.RunWithBootstrap(); err != nil {
		panic(err)
	}
}
`, moduleName, moduleName)

	return os.WriteFile(filepath.Join(projectPath, "main_bootstrap.go"), []byte(content), 0644)
}

func createGoMod(projectPath, moduleName string) error {
	content := fmt.Sprintf(`module %s

go 1.24.0
`, moduleName)

	return os.WriteFile(filepath.Join(projectPath, "go.mod"), []byte(content), 0644)
}

func createConfigYaml(projectPath string) error {
	content := `server:
  addr: "0.0.0.0"
  port: 8080
  read_timeout: 60
  write_timeout: 60
  use_default_routes: true

database:
  driver: "mysql"
  dsn: "root:password@tcp(localhost:3306)/database?charset=utf8mb4&parseTime=True&loc=Local"
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 3600

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 10

log:
  level: "info"
  format: "json"
  output: "stdout"
  max_size: 100
  max_backups: 3
  max_age: 28
`

	return os.WriteFile(filepath.Join(projectPath, "config.yaml"), []byte(content), 0644)
}

func createUtilityFiles(projectPath, moduleName string) error {
	configContent := `package config

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var globalConfig *Config

type Config struct {
	Server   ServerConfig   ` + "`yaml:\"server\"`" + `
	Database DatabaseConfig ` + "`yaml:\"database\"`" + `
	Redis    RedisConfig    ` + "`yaml:\"redis\"`" + `
	Log      LogConfig      ` + "`yaml:\"log\"`" + `
}

type ServerConfig struct {
	Host string ` + "`yaml:\"host\"`" + `
	Port int    ` + "`yaml:\"port\"`" + `
	Mode string ` + "`yaml:\"mode\"`" + `
}

type DatabaseConfig struct {
	Driver          string ` + "`yaml:\"driver\"`" + `
	Host            string ` + "`yaml:\"host\"`" + `
	Port            int    ` + "`yaml:\"port\"`" + `
	Username        string ` + "`yaml:\"username\"`" + `
	Password        string ` + "`yaml:\"password\"`" + `
	Database        string ` + "`yaml:\"database\"`" + `
	Charset         string ` + "`yaml:\"charset\"`" + `
	MaxIdleConns    int    ` + "`yaml:\"max_idle_conns\"`" + `
	MaxOpenConns    int    ` + "`yaml:\"max_open_conns\"`" + `
	ConnMaxLifetime int    ` + "`yaml:\"conn_max_lifetime\"`" + `
}

type RedisConfig struct {
	Host     string ` + "`yaml:\"host\"`" + `
	Port     int    ` + "`yaml:\"port\"`" + `
	Password string ` + "`yaml:\"password\"`" + `
	DB       int    ` + "`yaml:\"db\"`" + `
	PoolSize int    ` + "`yaml:\"pool_size\"`" + `
}

type LogConfig struct {
	Level      string ` + "`yaml:\"level\"`" + `
	Filename   string ` + "`yaml:\"filename\"`" + `
	MaxSize    int    ` + "`yaml:\"max_size\"`" + `
	MaxBackups int    ` + "`yaml:\"max_backups\"`" + `
	MaxAge     int    ` + "`yaml:\"max_age\"`" + `
	Compress   bool   ` + "`yaml:\"compress\"`" + `
}

type Component struct{}

func (c *Component) Name() string {
	return "config"
}

func (c *Component) Init(ctx context.Context) error {
	if globalConfig != nil {
		return nil
	}

	cfg := &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
			Mode: "debug",
		},
		Database: DatabaseConfig{
			Driver:          "mysql",
			Host:            "localhost",
			Port:            3306,
			Username:        "root",
			Password:        "",
			Database:        "test",
			Charset:         "utf8mb4",
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnMaxLifetime: 3600,
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			PoolSize: 10,
		},
		Log: LogConfig{
			Level:      "debug",
			Filename:   "logs/app.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
	}

	globalConfig = cfg
	return nil
}

func (c *Component) Close() error {
	return nil
}

type ConfigLoader struct{}

func (l *ConfigLoader) Load(path string) error {
	if globalConfig != nil {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %%w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %%w", err)
	}

	globalConfig = cfg
	return nil
}

func InitGlobalConfigWithConfig(config *Config) {
	globalConfig = config
}

func InitGlobalConfigWithPath(path string) error {
	loader := &ConfigLoader{}
	return loader.Load(path)
}

func InitGlobalConfigWithDefault() error {
	component := &Component{}
	return component.Init(context.Background())
}

func GetConfig() *Config {
	return globalConfig
}
`

	if err := os.WriteFile(filepath.Join(projectPath, "utility/config/config.go"), []byte(configContent), 0644); err != nil {
		return err
	}

	dbContent := fmt.Sprintf(`package db

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"%s/utility/config"
)

var globalDB *gorm.DB

type Component struct{}

func (c *Component) Name() string {
	return "db"
}

func (c *Component) Init(ctx context.Context) error {
	if globalDB != nil {
		return nil
	}

	cfg := config.GetConfig()
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}

	dsn := fmt.Sprintf("%%s:%%s@tcp(%%s:%%d)/%%s?charset=%%s&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
		cfg.Database.Charset,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %%w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %%w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)

	globalDB = db
	return nil
}

func (c *Component) Close() error {
	if globalDB != nil {
		sqlDB, err := globalDB.DB()
		if err == nil {
			return sqlDB.Close()
		}
	}
	return nil
}

func InitGlobalDBWithObject(db *gorm.DB) {
	globalDB = db
}

func InitGlobalDBWithConfig(cfg *config.DatabaseConfig) error {
	if globalDB != nil {
		return nil
	}

	dsn := fmt.Sprintf("%%s:%%s@tcp(%%s:%%d)/%%s?charset=%%s&parseTime=True&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %%w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %%w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	globalDB = db
	return nil
}

func InitGlobalDBWithDefault() error {
	component := &Component{}
	return component.Init(context.Background())
}

func GetDB() *gorm.DB {
	return globalDB
}
`, moduleName)

	if err := os.WriteFile(filepath.Join(projectPath, "utility/db/mysql.go"), []byte(dbContent), 0644); err != nil {
		return err
	}

	loggerContent := `package logger

import (
	"context"
	"os"

	"github.com/LandcLi/landc-go/log/facade"
	"gopkg.in/natefinch/lumberjack.v2"
)

var globalLogger facade.Logger

type Component struct{}

func (c *Component) Name() string {
	return "logger"
}

func (c *Component) Init(ctx context.Context) error {
	if globalLogger != nil {
		return nil
	}

	cfg := &facade.Config{
		Level:      "debug",
		Filename:   "logs/app.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	writer := &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	logger := facade.NewLogger()
	logger.SetOutput(writer)
	logger.SetLevel(cfg.Level)

	globalLogger = logger
	return nil
}

func (c *Component) Close() error {
	return nil
}

func InitGlobalLoggerWithObject(logger facade.Logger) {
	globalLogger = logger
}

func InitGlobalLoggerWithConfig(cfg *facade.Config) error {
	if globalLogger != nil {
		return nil
	}

	writer := &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	logger := facade.NewLogger()
	logger.SetOutput(writer)
	logger.SetLevel(cfg.Level)

	globalLogger = logger
	return nil
}

func InitGlobalLoggerWithDefault() error {
	component := &Component{}
	return component.Init(context.Background())
}

func GetLogger() facade.Logger {
	return globalLogger
}

func Debug(args ...interface{}) {
	globalLogger.Debug(args...)
}

func Info(args ...interface{}) {
	globalLogger.Info(args...)
}

func Warn(args ...interface{}) {
	globalLogger.Warn(args...)
}

func Error(args ...interface{}) {
	globalLogger.Error(args...)
}

func Fatal(args ...interface{}) {
	globalLogger.Fatal(args...)
	os.Exit(1)
}

func Debugf(format string, args ...interface{}) {
	globalLogger.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	globalLogger.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	globalLogger.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	globalLogger.Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	globalLogger.Fatalf(format, args...)
	os.Exit(1)
}
`

	if err := os.WriteFile(filepath.Join(projectPath, "utility/logger/logger.go"), []byte(loggerContent), 0644); err != nil {
		return err
	}

	validatorContent := `package validator

import (
	"context"

	"github.com/go-playground/validator/v10"
)

var globalValidator *validator.Validate

type Component struct{}

func (c *Component) Name() string {
	return "validator"
}

func (c *Component) Init(ctx context.Context) error {
	if globalValidator != nil {
		return nil
	}

	globalValidator = validator.New()
	return nil
}

func (c *Component) Close() error {
	return nil
}

func InitGlobalValidatorWithObject(v *validator.Validate) {
	globalValidator = v
}

func InitGlobalValidatorWithConfig(cfg *validator.Validate) error {
	if globalValidator != nil {
		return nil
	}

	globalValidator = cfg
	return nil
}

func InitGlobalValidatorWithDefault() error {
	component := &Component{}
	return component.Init(context.Background())
}

func GetValidator() *validator.Validate {
	return globalValidator
}

func Validate(s interface{}) error {
	return globalValidator.Struct(s)
}
`

	if err := os.WriteFile(filepath.Join(projectPath, "utility/validator/validator.go"), []byte(validatorContent), 0644); err != nil {
		return err
	}

	responseContent := `package response

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

var globalResponse *Response

type Response struct {
	Code    int         ` + "`json:\"code\"`" + `
	Message string      ` + "`json:\"message\"`" + `
	Data    interface{} ` + "`json:\"data\"`" + `
}

type Component struct{}

func (c *Component) Name() string {
	return "response"
}

func (c *Component) Init(ctx context.Context) error {
	if globalResponse != nil {
		return nil
	}

	globalResponse = &Response{
		Code:    0,
		Message: "success",
		Data:    nil,
	}
	return nil
}

func (c *Component) Close() error {
	return nil
}

func InitGlobalResponseWithObject(r *Response) {
	globalResponse = r
}

func InitGlobalResponseWithConfig(cfg *Response) error {
	if globalResponse != nil {
		return nil
	}

	globalResponse = cfg
	return nil
}

func InitGlobalResponseWithDefault() error {
	component := &Component{}
	return component.Init(context.Background())
}

func GetResponse() *Response {
	return globalResponse
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

func Fail(c *gin.Context, err error) {
	c.JSON(http.StatusOK, Response{
		Code:    -1,
		Message: err.Error(),
		Data:    nil,
	})
}
`

	if err := os.WriteFile(filepath.Join(projectPath, "utility/response/response.go"), []byte(responseContent), 0644); err != nil {
		return err
	}

	return nil
}

func createInternalFiles(projectPath, moduleName string) error {
	imptContent := fmt.Sprintf(`package internal

import (
	_ "%s/internal/controller"
)
`, moduleName)

	if err := os.WriteFile(filepath.Join(projectPath, "internal/impt.go"), []byte(imptContent), 0644); err != nil {
		return err
	}

	controllerContent := fmt.Sprintf(`package controller

import (
	"context"

	"github.com/LandcLi/landc-go/frame/pkg/meta"
	"%s/api/hello/v1"
	"%s/api/hello"
)

type HelloController struct {
	meta.Meta `+"`path:\"/hello\"`"+`
}

func init() {
	hello.RegisterHelloController(&HelloController{})
}

func (c *HelloController) SayHello(ctx context.Context, req *v1.SayHelloRequest) (*v1.SayHelloResponse, error) {
	return &v1.SayHelloResponse{
		Message: "Hello, " + req.Name + "!",
	}, nil
}
`, moduleName, moduleName)

	if err := os.WriteFile(filepath.Join(projectPath, "internal/controller/hello_controller.go"), []byte(controllerContent), 0644); err != nil {
		return err
	}

	return nil
}

func createApiFiles(projectPath, moduleName string) error {
	routersContent := fmt.Sprintf(`package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/LandcLi/landc-go/frame/pkg/web"
	"%s/api/hello"
)

func RegisterDefaultRouters(server *web.Server) {
	r := server.Engine()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Service is healthy",
		})
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	if err := server.RegisterHandler(hello.GetHelloController()); err != nil {
		panic(err)
	}
}
`, moduleName)

	if err := os.WriteFile(filepath.Join(projectPath, "api/routers.go"), []byte(routersContent), 0644); err != nil {
		return err
	}

	helloContent := fmt.Sprintf(`package hello

import (
	"context"

	"%s/api/hello/v1"
)

var globalHelloApi HelloApi

type HelloApi interface {
	SayHello(ctx context.Context, req *v1.SayHelloRequest) (*v1.SayHelloResponse, error)
}

func RegisterHelloController(controller HelloApi) {
	globalHelloApi = controller
}

func GetHelloController() HelloApi {
	return globalHelloApi
}
`, moduleName)

	if err := os.WriteFile(filepath.Join(projectPath, "api/hello/hello.go"), []byte(helloContent), 0644); err != nil {
		return err
	}

	sayHelloRequest := `package v1

import "github.com/LandcLi/landc-go/frame/pkg/meta"

type SayHelloRequest struct {
	meta.Meta ` + "`path:\"/say\" method:\"POST\" description:\"Say hello to user\"`" + `
	Name      string ` + "`json:\"name\" binding:\"required\"`" + `
}

type SayHelloResponse struct {
	Message string ` + "`json:\"message\"`" + `
}
`

	if err := os.WriteFile(filepath.Join(projectPath, "api/hello/v1/say_hello.go"), []byte(sayHelloRequest), 0644); err != nil {
		return err
	}

	return nil
}

func createReadme(projectPath, projectName string) error {
	content := fmt.Sprintf(`# %s

A landc-go project.

## Getting Started

### Prerequisites

- Go 1.24.0 or higher
- MySQL 5.7 or higher (optional)
- Redis 5.0 or higher (optional)

### Installation

1. Install dependencies:
   `+"```bash"+`
   go mod tidy
   `+"```"+`

2. Configure application:
   Edit `+"`config.yaml`"+` to set up your database and other settings.

3. Run application:
   `+"```bash"+`
   go run main.go
   `+"```"+`

## Bootstrap Auto-Initialization

This project uses Bootstrap for automatic initialization and lifecycle management. The Bootstrap will automatically:

- Load configuration from `+"`config.yaml`"+`
- Initialize internal components (config, logger, database, redis, validator, response)
- Execute lifecycle hooks (BeforeInit, AfterInit, BeforeRun, AfterRun)
- Clean up resources automatically on shutdown
- Support distributed tracing

## Project Structure

`+"```"+`
.
├── main.go              # Application entry point
├── config.yaml          # Configuration file
├── api/                 # API layer
│   ├── routers.go       # Route registration
│   └── hello/           # Example API
│       ├── hello.go     # API interface definition
│       └── v1/          # v1 version
│           └── say_hello.go  # Request/Response
├── service/             # Service layer interfaces
├── dao/                 # Data access layer interfaces
├── model/               # Data models
├── internal/            # Internal implementations
│   ├── controller/      # Controllers
│   ├── service/         # Service implementations
│   ├── dao/             # DAO implementations
│   └── impt.go          # Internal package initialization
└── test/                # Test files
`+"```"+`

## Development

### Adding a New Feature

1. Define API in `+"`api/`"+`
2. Create service interfaces in `+"`service/`"+`
3. Create DAO interfaces in `+"`dao/`"+`
4. Implement logic in `+"`internal/`"+`
5. Register routes in `+"`api/routers.go`"+`

### Bootstrap Configuration

The main.go file shows how to configure Bootstrap:

`+"```go"+`
import (
	"github.com/LandcLi/landc-go/frame/pkg/web"
)

func main() {
	server := web.NewServer(nil)
	
	server.Bootstrap().
		BeforeInit(func(ctx context.Context) error {
			fmt.Println("Initializing application...")
			return nil
		}).
		AfterInit(func(ctx context.Context) error {
			fmt.Println("Application initialized successfully")
			return nil
		}).
		BeforeRun(func(ctx context.Context) error {
			fmt.Println("Server starting...")
			return nil
		}).
		AfterRun(func(ctx context.Context) error {
			fmt.Println("Server stopped")
			return nil
		})

	if err := server.RunWithBootstrap(); err != nil {
		panic(err)
	}
}
`+"```"+`

### Adding Custom Components

If you need to add custom components to the initialization process:

`+"```go"+`
type MyComponent struct{}

func (c *MyComponent) Name() string {
	return "my-component"
}

func (c *MyComponent) Init(ctx context.Context) error {
	return nil
}

func (c *MyComponent) Close() error {
	return nil
}

// Then add it to Bootstrap:
server.Bootstrap().
	AddComponent(&MyComponent{})
`+"```"+`

## License

MIT License
`, projectName)

	return os.WriteFile(filepath.Join(projectPath, "README.md"), []byte(content), 0644)
}

func initGit(projectPath string) error {
	gitDir := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return nil
	}

	if err := os.MkdirAll(gitDir, 0755); err != nil {
		return err
	}

	return nil
}
