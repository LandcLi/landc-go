# Changelog

本项目所有重要变更都会记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [0.7.0] - 2026-08-07

### 新增

- **tools/security AES-GCM 加解密**：`Cipher`（`NewCipher`/`NewCipherFromString`，16/24/32 字节密钥，hex 或原始），`Encrypt`/`Decrypt`（crypto/rand 随机 nonce、`enc:` 前缀 + base64、无前缀明文兼容降级）；密钥由调用方传入，不绑定配置来源
- **tools/ratelimit 缓存限流**：`Cache` 接口（Get/Set/Delete/Exists + **原子 `Incr`**）+ `IntervalLimiter`（间隔限流）+ `CountLimiter`（计数限流），缓存由调用方注入，fail-open
- **tools/cache 原子自增**：`GlobalCache.Incr`（锁内原子读-改-写 + TTL 刷新）
- **tools/verifycode 验证码组件**：`Manager`（生成 + 发送间隔 60s + 每日上限 10 + TTL 5min 存储 + 一次性校验防重放），选项 `WithCodeLength`/`WithTTL`/`WithSendInterval`/`WithDailyLimit`，crypto/rand 数字码
- **frame/cache 增强**：`Cache` 接口新增 `Incr`（`RedisCache`：INCR + 首次创建设 TTL 避免窗口滑动；`LocalCache`：委托 tools）；`AsToolsCache` 适配器（frame Cache ↔ tools/ratelimit.Cache）
- **frame/pkg/ratelimit**（新包）：`AllowInterval(ctx, key, interval)` / `AllowCount(ctx, key, limit, window)`——缓存从请求 ctx 解析（`GetCacheFrom`），库模式嵌入自动使用命名缓存
- **frame/pkg/verifycode**（新包）：`Generate(ctx, key)` / `Verify(ctx, key, input)`——同上，缓存从 ctx 解析

### 破坏性变更

- `frame/pkg/cache.Cache` 接口新增 `Incr` 方法：框架内实现已同步（RedisCache / LocalCache）；外部自定义实现该接口的类型需补充该方法

## [0.6.0] - 2026-08-05

### 新增

- **web 注册选项**：`RegisterHandler(opts...)` 新增 `WithPrefix`（统一前缀）、`WithMethodPath`/`WithMethodHTTPMethod`（单方法路径/方法覆盖）、`WithMethodMiddleware`/`WithControllerMiddleware`（方法级/控制器级中间件，多个按序执行）——运行时选项优先于编译期 meta 标签，不传选项行为完全一致（下游本地集成需求 1+2）
- **路由查询**：`Server.Routes()` / `Server.Controllers()` 返回**最终生效**路由（Method/Path/Description/HandlerName），只读无副作用，供路由查询与文档生成共用（需求 4）
- **proxy 显式构造**：生成代码新增 `NewXXXProxy(baseURL)` 构造函数（与 `ProvideRemote` 等价），原 `init()` + blank import 用法保留兼容（需求 5）
- **OpenAPI 自动收集**：`openapi.RegisterServer(server)` 从已注册控制器生成文档（路径为最终生效路由）、`RegisterControllerRoutes`、`WriteFile`（落盘 openapi.json）——供 CI 生成 API 文档（需求 6）
- **命名资源作用域**：`resource.Scope` + `config.InitNamedConfig` / `db.InitNamedDB` / `cache.InitNamedCache(WithConfig|WithLocal)` + `GetXxxFrom(ctx)`；`web.RegisterLibrary` 注册时校验命名资源并自动注入作用域中间件——库模式嵌入服务可使用宿主为之准备的**独立** DB/缓存/配置，未指定字段回退全局（不静默回退，防数据写错库）
- **`landc migrate-dbctx` 命令**：自动为 DAO/service 接口与方法注入 `ctx context.Context` 参数，迁移 `db.GetDB()`→`db.GetDBFrom(ctx)`、`cache.GetCache()`→`cache.GetCacheFrom(ctx)`、`context.Background()`→`ctx`
- **`landc gen lib` 命令**：生成库模式入口 `serverlib/RegisterToRouter` 骨架（Gateway 模式 + `WithAuth`/`WithWebOptions` + 防御性 bootstrap 检查）
- **`db.GetTx` / `Transaction` 作用域感知**：连接从 ctx 解析（`GetDBFrom`），生成 DAO 自动支持命名资源
- **web handler ctx 语义**：controller 第一参数为 `context.Context` 接口时传入携带资源作用域的请求 ctx（`*gin.Context` 用法保留）

### 修复

- **migrate-dbctx 误迁移**：仅给方法体访问资源（GetDB/GetCache/context.Background）的方法加 ctx，避免误改辅助方法；删除 `ctx := context.Background()` 赋值（防 `ctx := ctx` 自引用）
- **`landc gen lib` 不可用**：`pkg/gen` 子命令经 `GenSubcommands()` 正确挂载（原嵌套成 `gen gen`）
- **proxygen 接口包路径推导**：`filepath.Rel` 前统一转为绝对路径，修复生成 `import "module/"` 尾部斜杠 malformed
- **gen proxy 参数提示**：统一为 `--type`（解析器仅识别 `--` 前缀）

### 破坏性变更

- 无（所有新能力不显式使用时行为与 v0.5.0 完全一致；现有全局单例 API 保留）

## [0.5.0] - 2026-08-04

### 新增

- **JWT 支持非对称签名**：`JWTConfig` 新增 `SigningMethod`（HS256/RS256/ES256）、`PrivateKeyPath`、`PublicKeyPath` 及可编程注入的 `PrivateKey`/`PublicKey`；`ParseToken` 通过算法白名单（`WithValidMethods`）防御算法混淆攻击；PEM 密钥按路径缓存加载；未配置 `SigningMethod` 时默认 HS256，向后兼容
- **workflow ScriptExecutor 实现 JS 执行**：基于 goja（纯 Go，无 CGO），支持 `input`/`inputRaw` 注入与超时中断；不支持的语言返回明确错误
- **condition 节点 Expression 模式实现**：基于 goja 的 JS 布尔表达式求值（`input` 变量注入），替代原先的预留桩
- **工作流测试全量补齐**：8 个包全部有测试（engine/executor/idempotent/store/scheduler/lock/observer/model），含 `-race` 并发验证
- **锁内存实现 `MemoryLock`**：进程内、token 校验、TTL + 自动续约，语义与 RedisLock 一致，供无 Redis 单机/测试场景
- **框架全链路集成测试**（`frame/tests/`）：Config(YAML) → DB(sqlite) → DI → Controller(Meta Tag 路由) → 中间件 → JWT → 统一响应，6 个用例
- **workflow + saas 跨模块集成测试**（`workflow/pkg/integration/`）：引擎执行时节点调用 saas `TenantScope` 做租户数据隔离校验
- **并发压力冒烟测试**（`engine_stress_test.go`）：50 并发执行 + 60 节点 DAG ×5 轮
- **性能基准**：JWT 签发/验签、内存 store、saas scope 查询（记录于 `docs/benchmarks.md`）
- **示例补齐**：`log/`、`tools/`、`saas/` 新增 `examples/`（已验证可编译可运行）
- **CI 与工程化**：GitHub Actions 流水线（lint/test/build/security 四 job）、`.golangci.yml`、`Makefile`、根 `.gitignore`、`go.work` 工作区、`Dockerfile`（builder/dev/production 三阶段）、`.env.example`、`CONTRIBUTING.md`、`CODE_OF_CONDUCT.md`、`SECURITY.md`、`.goreleaser.yaml`、`CHANGELOG.md`

### 修复

- **`executeDAG` 数据竞争**：主循环无锁读 `completedNodes`，与 batch goroutine 并发写冲突（由压力冒烟测试发现），加 `completedMu` 保护
- **`saas.ListTenantData`**：内存分页改为 SQL 层 UNION + LIMIT/OFFSET 分页
- **`saas.TenantScopeWithAccess` 使用 `NOW()`**：不可移植到 sqlite，改为参数化 `time.Now()`
- **`saas.GetTenantTree` 子树构建失败**：`rootID=nil` 时只查根租户导致 children 为空；子树场景顶层节点匹配 `root.ParentID`
- **`workflow/init.go`**：移除无效占位行（`_ = ttl`、`_ = trace.TraceID`）；`InitWithComponents` 改为真正使用传入的 `*gorm.DB`
- **`SubWorkflowExecutor`**：从静默透传改为明确报错（避免"宣称支持但无实现"）
- **api 模块 GoFrame 依赖过重**：`middleware/goframe` 及示例改为 `-tags goframe` 可选编译
- **JWT 配置热更新不同步**：bootstrap 新增 `WatchJWTConfig` 自动同步，已接入生命周期
- **saas `DataAccess.CheckConstraint` 实现**：消除"宣称校验但直接返回 true"的预留桩，非法约束保守拒绝（fail-closed）
- **workflow README 能力矩阵对齐**：SCRIPT 标注 JS（goja）、SUB_WORKFLOW 明确标注"规划中"
- **golangci-lint 全量清零**：6 模块 0 error，修复 80+ 处（含 converter 溢出检查缺失、tag 负数阈值绕过 2 个真实缺陷）
- **`MemoryIdempotencyChecker` 并发写 map 数据竞争**：加 RWMutex
- **DI 容器 `RegisterLazySingleton`**：返回闭包而非实例的功能性 Bug
- **`Command.Run` 对 nil Bootstrap**：空指针 panic 修复
- **workflow 引擎共享锁**：无意义的局部 `sync.Mutex` 改为共享锁保护 `completedNodes`
- **web 测试断言**：与统一响应格式不匹配修复

### 安全

- **Go 1.24 → 1.26 升级**（go 1.24 已于 2026-02 停止安全支持）：统一所有 go.mod/go.work 的 `go` 指令为 `1.26`，修复标准库（crypto/tls、x509、net 等 8 项）漏洞
- **修复依赖漏洞**（govulncheck v1.6.0 全模块 0 漏洞）：grpc v1.72.0 → v1.82.1（GO-2026-6061）、`golang.org/x/text` → v0.39.0（GO-2026-5970）、pgx v5.6.0 → v5.9.2、quic-go v0.54.0 → v0.59.1、`golang.org/x/sys` → v0.46.0
- **golangci-lint v1.64.8 → v2.12.2**：配置迁移至 v2 格式，修复 v2 新增检查 60+ 处
- **JWT 私钥权限强制 0600**：RS256/ES256 私钥文件组/其他用户可读时拒绝加载
- **`saas.RevokeAccess` 越权漏洞**：撤销共享前校验当前租户为数据 owner
- **workflow HTTP 节点 SSRF**：默认拒绝内网/保留地址目标，需显式开启 `allow_private_network`
- **JWT 默认密钥硬编码**：改为环境变量 `LANDC_JWT_SECRET` 注入；密钥不足 32 字符拒绝签发/解析
- **WebSocket CSWSH**：默认同源校验
- **文件上传路径穿越**：`FileNameFunc` 返回值经 `filepath.Base` 消毒
- **CORS**：默认 `*` 与 `Allow-Credentials` 非法共存修复
- **内部错误透传**：统一返回 `internal server error` 并记录日志
- **`TenantScopeWithConstraint` 静默放行**：实现约束验证并拒绝非法约束
- **DB 默认日志级别**：`Info` → `Warn`，可通过 `LANDC_DB_LOG_MODE` 覆盖

### 文档

- `frame/README.md` 新增"安全行为"小节（JWT 算法/密钥长度/私钥权限、CORS 凭据策略、WebSocket 同源）
- 新增 `docs/releases.md`（发布流程、tag 规范、回滚方案）、`docs/dependencies.md`（依赖基线）、`docs/benchmarks.md`（性能基线）、`docs/licenses.csv`（第三方许可证清单）
- `CONTRIBUTING.md` 新增版本与发布小节；workflow/saas/api README 能力矩阵对齐

### 发布

- **release.yml 多模块重写**：支持 `log/v*`、`tools/v*`、`api/v*`、`frame/v*`、`workflow/v*`、`saas/v*`、`v*` 触发；`validate-tag` job 强制严格 semver + 模块前缀白名单；发布前 6 模块 build+test
- **`scripts/release-check.sh`**：发布前冻结检查自动化（git 干净 / build / vet / test / lint / govulncheck / mod verify / go 指令一致性）
- 第三方许可证审计：78 个依赖无 GPL/AGPL/SSPL（唯一非 MIT 为 MPL-2.0 可接受）

## [0.1.0] - 2026-07-25

### 新增

- **log**：日志门面，支持 Zap / Logrus / Console 三后端，gin/goframe 适配器
- **tools**：缓存（LRU+过期）、字符串、数字、UUID/雪花 ID、地理定位、邮箱验证、SMTP 邮件、Tag 验证、HTTP 客户端、限流、安全工具
- **api**：统一 Response / ErrorCode 体系、Gin/GoFrame 中间件、链路追踪
- **frame**：Web 引擎（Meta Tag 路由）、DI 容器、配置管理、GORM DB、Redis、JWT、中间件、i18n、session、websocket、cron、openapi、registry、gRPC、代码生成、trace、bootstrap、CLI
- **workflow**：分布式工作流引擎（DAG 编排、节点执行、状态管理、调度器、幂等、分布式锁）
- **saas**：多租户插件（租户、数据归属、访问控制、共享日志）
