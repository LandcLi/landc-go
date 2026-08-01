# Changelog

本项目所有重要变更都会记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 新增

- **JWT 支持非对称签名**：`JWTConfig` 新增 `SigningMethod`（HS256/RS256/ES256）、`PrivateKeyPath`、`PublicKeyPath` 及可编程注入的 `PrivateKey`/`PublicKey`；`ParseToken` 通过算法白名单（`WithValidMethods`）防御算法混淆攻击；PEM 密钥按路径缓存加载；未配置 `SigningMethod` 时默认 HS256，向后兼容
- **workflow ScriptExecutor 实现 JS 执行**：基于 goja（纯 Go，无 CGO），支持 `input`/`inputRaw` 注入与超时中断；不支持的语言返回明确错误
- workflow executor 测试（脚本执行、超时、SSRF 地址识别等）

### 安全

- **修复**：`saas.RevokeAccess` 越权漏洞——撤销共享前校验当前租户为数据 owner，非 owner 撤销被拒绝
- **修复**：workflow HTTP 节点 SSRF——默认拒绝内网/保留地址目标，可通过 `allow_private_network` 显式开启
- 升级 `google.golang.org/grpc` v1.62.2 → v1.83.0（安全敏感组件）

### 修复

- `workflow/init.go`：移除无效占位行（`_ = ttl`、`_ = trace.TraceID`）；`InitWithComponents` 改为真正使用传入的 `*gorm.DB`
- `saas.ListTenantData`：内存分页改为 SQL 层 UNION + LIMIT/OFFSET 分页
- 清理 `saas/go.mod` 冗余 replace 死配置；统一 Go 版本格式
- `.gitignore`：移除对 `go.work`/`go.work.sum` 的忽略（仓库实际跟踪）
- `SECURITY.md` / `CODE_OF_CONDUCT.md` 占位符替换为实际联系邮箱
- `SubWorkflowExecutor` 从静默透传改为明确报错（避免"宣称支持但无实现"）

- 添加 GitHub Actions CI 流水线（lint / test / build / security 四 job）
- 添加 `.golangci.yml`（gofmt / govet / staticcheck / errcheck / gosec / gocyclo 等）
- 添加 `Makefile` 统一构建、测试、Lint 命令
- 添加根目录 `.gitignore`
- 添加 `go.work` 工作区文件，支持本地多模块开发
- 添加 `Dockerfile`（builder / dev / production 三阶段）
- 添加 `.env.example` 环境变量模板
- 添加社区文件：`CONTRIBUTING.md`、`CODE_OF_CONDUCT.md`、`SECURITY.md`
- 添加 `.goreleaser.yaml` 与 release 发布流水线
- 添加 `CHANGELOG.md`

### 安全

- **修复**：saas 模块 import 路径错误导致无法编译（`landc-go/saas/...` → `github.com/LandcLi/landc-go/saas/...`）
- **修复**：JWT 默认密钥 `"change-me"` 硬编码，改为从环境变量 `LANDC_JWT_SECRET` 注入；密钥长度不足 32 字符时拒绝签发/解析
- **修复**：WebSocket 默认允许任意跨域 Origin（CSWSH 风险），改为默认同源校验
- **修复**：文件上传自定义 `FileNameFunc` 返回值未做路径穿越防护，新增 `filepath.Base` 消毒
- **修复**：CORS 默认 `*` 与 `Allow-Credentials` 非法共存，现在仅在显式配置来源时允许凭据
- **修复**：内部错误直接透传客户端，改为统一返回 `internal server error` 并记录日志
- **修复**：`TenantScopeWithConstraint` 中带约束的访问记录被静默放行（越权风险），实现约束验证并拒绝非法约束
- **修复**：DB 默认日志级别从 `Info` 改为 `Warn`，可通过 `LANDC_DB_LOG_MODE` 覆盖

### 修复

- **修复**：DI 容器 `RegisterLazySingleton` 返回闭包而非实例的功能性 Bug
- **修复**：`Command.Run` 对 `Bootstrap` 为 nil 时的空指针 panic
- **修复**：workflow 引擎中无意义的局部 `sync.Mutex`，改为共享锁保护 `completedNodes` 并发写
- **修复**：`frame/examples` 中 `package main` 缺少 `main()` 导致的编译失败
- **修复**：`tools/tag` 中两处不可达代码
- **修复**：`web` 测试断言与统一响应格式不匹配

## [0.1.0] - 2026-07-25

### 新增

- **log**：日志门面，支持 Zap / Logrus / Console 三后端，gin/goframe 适配器
- **tools**：缓存（LRU+过期）、字符串、数字、UUID/雪花 ID、地理定位、邮箱验证、SMTP 邮件、Tag 验证、HTTP 客户端、限流、安全工具
- **api**：统一 Response / ErrorCode 体系、Gin/GoFrame 中间件、链路追踪
- **frame**：Web 引擎（Meta Tag 路由）、DI 容器、配置管理、GORM DB、Redis、JWT、中间件、i18n、session、websocket、cron、openapi、registry、gRPC、代码生成、trace、bootstrap、CLI
- **workflow**：分布式工作流引擎（DAG 编排、节点执行、状态管理、调度器、幂等、分布式锁）
- **saas**：多租户插件（租户、数据归属、访问控制、共享日志）
