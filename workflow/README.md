# Workflow — 分布式工作流引擎

基于 landc-go 框架的 DAG 工作流引擎，支持分布式调度、暂停/恢复、可重入、幂等执行和全链路可观测。

---

## 目录

- [快速开始](#快速开始)
- [核心概念](#核心概念)
- [配置说明](#配置说明)
- [使用指南](#使用指南)
  - [方式一：配置驱动（推荐）](#方式一配置驱动推荐)
  - [方式二：手动组装](#方式二手动组装)
  - [方式三：嵌入已有服务](#方式三嵌入已有服务)
- [管理 API](#管理-api)
- [编程式使用 Engine](#编程式使用-engine)
- [工作流定义](#工作流定义)
- [节点类型](#节点类型)
- [DAG 调度](#dag-调度)
- [分布式调度器](#分布式调度器)
- [可观测性](#可观测性)
- [幂等性](#幂等性)
- [暂停/恢复与可重入](#暂停恢复与可重入)
- [重试策略](#重试策略)
- [Key 设计](#key-设计)
- [常见问题](#常见问题)

---

## 快速开始

### 前置要求

- Go 1.26+
- landc-go 框架（`frame`, `log`, `tools`）
- MySQL / PostgreSQL / SQLite（任一）
- Redis（可选，用于分布式锁和幂等性增强）
- etcd（可选，用于分布式 Worker 注册）

### 最小示例

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"

	// 1. 导入 internal 包触发所有 init() 自动注册
	_ "github.com/LandcLi/landc-go/workflow/internal"

	// 2. 导入 workflow 框架
	"github.com/LandcLi/landc-go/workflow"
	"github.com/LandcLi/landc-go/workflow/pkg/engine"
	"github.com/LandcLi/landc-go/workflow/pkg/executor"
	"github.com/LandcLi/landc-go/workflow/pkg/idempotent"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
	"github.com/LandcLi/landc-go/workflow/pkg/observer"
	"github.com/LandcLi/landc-go/workflow/pkg/store"

	// 3. 框架初始化
	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/LandcLi/landc-go/frame/pkg/db"
	"github.com/LandcLi/landc-go/log/facade"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// ===== 第一步：初始化数据库 =====
	gormDB, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	_ = db.InitGlobalDBWithObject(gormDB)

	// ===== 第二步：初始化工作流框架 =====
	workflow.Init()

	// ===== 第三步：通过 DI 获取引擎，直接使用 =====
	eng := engine.GetEngine()

	// ===== 第四步：创建工作流定义 =====
	wf := &model.Workflow{
		ID:          "wf-demo-001",
		Name:        "Demo 审批流程",
		Description: "一个简单的 DAG 审批工作流",
		Version:     1,
		Status:      model.WorkflowStatusActive,
		Timeout:     3600, // 1小时超时
		Nodes: []*model.Node{
			{
				ID: "node-start", WorkflowID: "wf-demo-001", Name: "参数校验",
				Type: model.NodeTypeHttp, OrderNo: 0,
				Config: json.RawMessage(`{"url":"http://svc/validate","method":"POST"}`),
			},
			{
				ID: "node-review", WorkflowID: "wf-demo-001", Name: "人工审批",
				Type: model.NodeTypeHttp, OrderNo: 1,
				MaxRetries: 3, RetryDelay: 5, RetryMode: "EXPONENTIAL",
				Config: json.RawMessage(`{"url":"http://svc/review","method":"POST"}`),
			},
			{
				ID: "node-notify", WorkflowID: "wf-demo-001", Name: "结果通知",
				Type: model.NodeTypeHttp, OrderNo: 2,
				Config: json.RawMessage(`{"url":"http://svc/notify","method":"POST"}`),
			},
		},
		Edges: []*model.Edge{
			{ID: "e1", WorkflowID: "wf-demo-001", SourceID: "node-start", TargetID: "node-review"},
			{ID: "e2", WorkflowID: "wf-demo-001", SourceID: "node-review", TargetID: "node-notify"},
		},
	}
	store.GetStore().CreateWorkflow(context.Background(), wf)

	// ===== 第五步：启动工作流 =====
	input, _ := json.Marshal(map[string]interface{}{
		"applicant": "张三",
		"amount":    5000,
	})
	execID, _ := eng.StartWorkflow(context.Background(), wf.ID, input,
		model.TriggerTypeApi, "req-001")

	fmt.Printf("工作流已启动，执行ID: %s\n", execID)

	// ===== 第六步：查询执行状态 =====
	exec, _ := eng.GetExecutionStatus(context.Background(), execID)
	fmt.Printf("状态: %s\n", exec.Status)

	// ===== 第七步：暂停/恢复 =====
	eng.PauseWorkflow(context.Background(), execID)  // 暂停
	eng.ResumeWorkflow(context.Background(), execID)  // 恢复（可重入）

	// ===== 第八步：取消 =====
	eng.CancelWorkflow(context.Background(), execID)  // 取消执行
}
```

---

## 核心概念

| 概念 | 说明 |
|------|------|
| **Workflow** | 工作流定义，包含 DAG 的节点（Nodes）和边（Edges） |
| **Node** | DAG 中的一个执行步骤，每个节点有类型、配置和重试策略 |
| **Edge** | 定义节点间的依赖关系，支持条件边（Label + ConditionExpr） |
| **Execution** | 一次工作流运行实例，记录整个执行生命周期 |
| **Task** | 一个节点的单次执行记录，包含重试计数和幂等标识 |
| **DAG** | 有向无环图，引擎按拓扑排序依次执行节点 |
| **Reentrant** | 可重入：已完成的节点自动跳过，从中断处继续 |
| **Idempotent** | 幂等：相同的 TriggerID 不会重复创建执行 |

### 执行状态机

```
PENDING ──► RUNNING ──► COMPLETED
              │
              ├──► FAILED
              ├──► CANCELLED
              ├──► TIMEOUT
              └──► PAUSED ──► RUNNING (可重入恢复)
```

---

## 配置说明

### config.yaml 配置项

```yaml
server:
  addr: "0.0.0.0"
  port: 8080

database:
  dsn: "root:pass@tcp(127.0.0.1:3306)/workflow?charset=utf8mb4&parseTime=True"
  max_open_conns: 100
  max_idle_conns: 10

redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 0

# 工作流框架专属配置（放在 other 下）
other:
  workflow:
    engine:
      max_parallel_tasks: 10    # 每个执行的最大并行节点数
      default_timeout: 1800      # 默认超时时间（秒）
      idempotency_ttl: 86400     # 幂等 Key 过期时间（秒）
    scheduler:
      poll_interval: 2           # 任务轮询间隔（秒）
      batch_size: 20             # 每批拉取的任务数
```

配置通过 `frame/pkg/config.InitGlobalConfigWithPath("config.yaml")` 加载后，
调用 `workflow.Init()` 会自动提取上述配置。

---

## 使用指南

### 方式一：配置驱动（推荐）

完整的服务化部署方式，适用于生产环境。

```go
// main.go
package main

import (
	"context"
	"log"

	// 触发所有 init() 自动注册 Controller / Service / DAO
	_ "github.com/LandcLi/landc-go/workflow/internal"

	"github.com/LandcLi/landc-go/workflow"
	"github.com/LandcLi/landc-go/workflow/internal/cmd"
	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/LandcLi/landc-go/frame/pkg/db"
	framecache "github.com/LandcLi/landc-go/frame/pkg/cache"
)

func main() {
	// 1. 加载配置文件
	config.InitGlobalConfigWithPath("config.yaml")

	// 2. 初始化 DB（自动从配置读取 DSN）
	db.InitGlobalDB()

	// 3. 初始化缓存（Redis 可用则用 Redis，否则回退本地 LRU）
	framecache.InitGlobalCacheWithDefault()

	// 4. 初始化工作流框架
	if err := workflow.Init(); err != nil {
		log.Fatalf("workflow init failed: %v", err)
	}

	// 5. 启动 HTTP 服务（自动注册管理 API 路由）
	if err := cmd.Main.Run(context.Background()); err != nil {
		log.Fatalf("server start failed: %v", err)
	}
}
```

启动后，管理 API 自动可用：

```bash
# 创建工作流
curl -X POST http://localhost:8080/api/workflows \
  -H 'Content-Type: application/json' \
  -d '{"name":"审批流程","nodes":[...],"edges":[...]}'

# 启动工作流
curl -X POST http://localhost:8080/api/workflows/{id}/start \
  -H 'Content-Type: application/json' \
  -d '{"trigger_id":"req-001","input":"{\"key\":\"value\"}"}'

# 暂停执行
curl -X POST http://localhost:8080/api/executions/{id}/pause

# 恢复执行
curl -X POST http://localhost:8080/api/executions/{id}/resume

# 查询执行
curl http://localhost:8080/api/executions/{id}

# 查询任务列表
curl http://localhost:8080/api/executions/{id}/tasks

# 工作流列表
curl http://localhost:8080/api/workflows?page=1&size=20
```

### 方式二：手动组装

适用于测试、快速原型或不想依赖框架配置的场景。

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LandcLi/landc-go/workflow/pkg/engine"
	"github.com/LandcLi/landc-go/workflow/pkg/executor"
	"github.com/LandcLi/landc-go/workflow/pkg/idempotent"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
	"github.com/LandcLi/landc-go/workflow/pkg/observer"
	"github.com/LandcLi/landc-go/workflow/pkg/store"
	"github.com/LandcLi/landc-go/tools/generate"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// ---- 初始化 ----
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	dbStore := store.NewDBStore(db)
	dbStore.AutoMigrate()

	execReg := executor.NewRegistry()
	executor.RegisterDefault(execReg)

	obsManager := observer.NewObserverManager()
	obsManager.Register(observer.NewLogObserver(nil))

	eng := engine.NewEngine(
		dbStore, execReg, obsManager,
		idempotent.NewMemoryIdempotencyChecker(24*time.Hour),
		engine.DefaultEngineConfig(),
	)

	// ---- 创建工作流定义 ----
	wfID := generate.UUID()
	wf := &model.Workflow{
		ID: wfID, Name: "数据管道", Status: model.WorkflowStatusActive,
		Nodes: []*model.Node{
			{ID: "n1", WorkflowID: wfID, Name: "数据抽取", Type: model.NodeTypeHttp, OrderNo: 0,
				Config: json.RawMessage(`{"url":"http://etl/extract","method":"POST"}`)},
			{ID: "n2", WorkflowID: wfID, Name: "数据转换", Type: model.NodeTypeHttp, OrderNo: 1,
				Config: json.RawMessage(`{"url":"http://etl/transform","method":"POST"}`)},
			{ID: "n3", WorkflowID: wfID, Name: "数据加载", Type: model.NodeTypeHttp, OrderNo: 2,
				Config: json.RawMessage(`{"url":"http://etl/load","method":"POST"}`)},
		},
		Edges: []*model.Edge{
			{ID: "e1", WorkflowID: wfID, SourceID: "n1", TargetID: "n2"},
			{ID: "e2", WorkflowID: wfID, SourceID: "n2", TargetID: "n3"},
		},
	}
	dbStore.CreateWorkflow(context.Background(), wf)

	// ---- 启动 ----
	input, _ := json.Marshal(map[string]string{"source": "mysql"})
	execID, err := eng.StartWorkflow(context.Background(), wfID, input,
		model.TriggerTypeApi, "")
	if err != nil {
		panic(err)
	}
	fmt.Println("Execution ID:", execID)

	// ---- 等待完成 ----
	for i := 0; i < 20; i++ {
		exec, _ := eng.GetExecutionStatus(context.Background(), execID)
		if exec.IsFinal() {
			fmt.Println("Status:", exec.Status)
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// ---- 查看执行详情 ----
	tasks, _ := eng.GetExecutionTasks(context.Background(), execID)
	for _, t := range tasks {
		fmt.Printf("  [%s] %s (重试: %d)\n", t.Status, t.NodeName, t.RetryCount)
	}
}
```

### 方式三：嵌入已有服务

如果已经有一个正在运行的 landc-go 服务，可以直接把 workflow 的 controller 注册进去。

```go
package main

import (
	"context"

	_ "github.com/LandcLi/landc-go/workflow/internal"                // 触发 init() 注册
	workflowapi "github.com/LandcLi/landc-go/workflow/api"           // Controller + Gateway
	"github.com/LandcLi/landc-go/frame/pkg/web"                     // web server
	"github.com/LandcLi/landc-go/frame/pkg/config"                  // config
	"github.com/LandcLi/landc-go/frame/pkg/db"                      // db
	framecache "github.com/LandcLi/landc-go/frame/pkg/cache"         // cache
	"github.com/LandcLi/landc-go/workflow"                          // Init
)

func main() {
	config.InitGlobalConfigWithPath("config.yaml")
	db.InitGlobalDB()
	framecache.InitGlobalCacheWithDefault()
	workflow.Init()

	// 既有 controller + workflow controller 一起注册
	server := web.New()
	server.RegisterHandler(yourExistingController)
	server.RegisterHandler(workflowapi.GetWorkflowController())
	server.RegisterHandler(workflowapi.GetExecutionController())
	server.RunWithContext(context.Background())
}
```

---

## 管理 API

框架提供了完整的管理 REST API，通过 `api/` 层的 Controller + Gateway 自动注册到 Gin。

### Workflow（工作流管理）

| 方法 | 路径 | 说明 | 请求体 |
|------|------|------|--------|
| POST | `/api/workflows` | 创建工作流定义 | `CreateWorkflowRequest` |
| GET | `/api/workflows` | 工作流列表（分页） | `?page=1&size=20` |
| GET | `/api/workflows/:id` | 查询单个工作流 | — |
| DELETE | `/api/workflows/:id` | 删除工作流 | — |
| POST | `/api/workflows/:id/start` | 启动工作流执行 | `{trigger_id, input}` |

### Execution（执行管理）

| 方法 | 路径 | 说明 | 请求体 |
|------|------|------|--------|
| GET | `/api/executions` | 执行列表（分页） | `?workflow_id=&status=&page=1&size=20` |
| GET | `/api/executions/:id` | 查询执行详情 | — |
| POST | `/api/executions/:id/pause` | 暂停执行 | — |
| POST | `/api/executions/:id/resume` | 恢复执行 | — |
| POST | `/api/executions/:id/cancel` | 取消执行 | — |
| GET | `/api/executions/:id/tasks` | 任务列表 | — |

### API 调用示例

```bash
# 创建 DAG 工作流
curl -X POST http://localhost:8080/api/workflows \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "订单处理",
    "nodes": [
      {"id":"n1","name":"验单","type":"HTTP","order_no":0,"config":"{\"url\":\"http://order/validate\"}"},
      {"id":"n2","name":"支付","type":"HTTP","order_no":1,"config":"{\"url\":\"http://order/pay\"}"},
      {"id":"n3","name":"发货","type":"HTTP","order_no":2,"config":"{\"url\":\"http://order/ship\"}"},
      {"id":"n4","name":"通知","type":"HTTP","order_no":3,"config":"{\"url\":\"http://order/notify\"}"}
    ],
    "edges": [
      {"id":"e1","source_id":"n1","target_id":"n2"},
      {"id":"e2","source_id":"n2","target_id":"n3"},
      {"id":"e3","source_id":"n2","target_id":"n4","label":"rejected"}
    ]
  }'

# 启动
curl -X POST http://localhost:8080/api/workflows/{wf_id}/start \
  -H 'Content-Type: application/json' \
  -d '{"trigger_id":"order-123","input":"{\"order_id\":\"ORD-2024-001\"}"}'

# 暂停/恢复
curl -X POST http://localhost:8080/api/executions/{exec_id}/pause
curl -X POST http://localhost:8080/api/executions/{exec_id}/resume
```

---

## 编程式使用 Engine

如果不通过 HTTP API，也可以直接编程式使用 Engine：

```go
import "github.com/LandcLi/landc-go/workflow/pkg/engine"

eng := engine.GetEngine()

// 启动（幂等：相同的 triggerID 不会重复创建）
execID, err := eng.StartWorkflow(ctx, workflowID, input, model.TriggerTypeApi, "my-trigger-id")

// 暂停
eng.PauseWorkflow(ctx, execID)

// 恢复（可重入：已完成节点自动跳过）
eng.ResumeWorkflow(ctx, execID)

// 取消
eng.CancelWorkflow(ctx, execID)

// 查询
exec, _ := eng.GetExecutionStatus(ctx, execID)
tasks, _ := eng.GetExecutionTasks(ctx, execID)

// 遍历任务
for _, t := range tasks {
    fmt.Printf("%s: %s (retry: %d)\n", t.NodeName, t.Status, t.RetryCount)
}
```

---

## 工作流定义

一个完整的工作流定义包含 Node（节点）和 Edge（边）。

### Node 字段说明

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `ID` | string | 是 | 节点唯一标识 |
| `Name` | string | 是 | 节点名称，用于可观测性 |
| `Type` | NodeType | 是 | HTTP / SCRIPT / DELAY / CONDITION 等 |
| `Config` | JSON | 否 | 节点配置，按 Type 不同而不同 |
| `Timeout` | int64 | 否 | 节点超时（秒） |
| `MaxRetries` | int | 否 | 失败重试次数，默认 0 |
| `RetryDelay` | int64 | 否 | 重试间隔（秒） |
| `RetryMode` | string | 否 | `LINEAR` / `EXPONENTIAL`，默认 LINEAR |
| `InputMapping` | JSON | 否 | 上游输出到输入的映射 |
| `OrderNo` | int | 否 | 排序号 |

### Edge 字段说明

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `ID` | string | 是 | 边唯一标识 |
| `SourceID` | string | 是 | 上游节点 ID |
| `TargetID` | string | 是 | 下游节点 ID |
| `Label` | string | 否 | 边标签（如 "approved"、"rejected"） |
| `ConditionExpr` | string | 否 | 条件表达式，为空时无条件通过 |

### 完整定义示例

```go
wf := &model.Workflow{
    ID:          "wf-order",
    Name:        "订单处理流程",
    Description: "从验单到发货的完整流程",
    Version:     1,
    Status:      model.WorkflowStatusActive,
    Timeout:     7200,
    MaxRetries:  3,
    Nodes: []*model.Node{
        {
            ID: "step-validate", WorkflowID: "wf-order",
            Name: "验单", Type: model.NodeTypeHttp,
            Config: json.RawMessage(`{"url":"http://order/validate","method":"POST","timeout":30}`),
            MaxRetries: 2, RetryDelay: 5, RetryMode: "LINEAR",
        },
        {
            ID: "step-pay", WorkflowID: "wf-order",
            Name: "支付", Type: model.NodeTypeHttp,
            Config: json.RawMessage(`{"url":"http://order/pay","method":"POST"}`),
            MaxRetries: 3, RetryDelay: 10, RetryMode: "EXPONENTIAL",
        },
        {
            ID: "step-ship", WorkflowID: "wf-order",
            Name: "发货", Type: model.NodeTypeHttp,
            Config: json.RawMessage(`{"url":"http://order/ship","method":"POST"}`),
        },
        {
            ID: "step-notify", WorkflowID: "wf-order",
            Name: "通知", Type: model.NodeTypeHttp,
            Config: json.RawMessage(`{"url":"http://order/notify","method":"POST"}`),
        },
    },
    Edges: []*model.Edge{
        // 验单通过 → 支付
        {ID: "e-v-p", WorkflowID: "wf-order", SourceID: "step-validate", TargetID: "step-pay", Label: "approved"},
        // 验单不通过 → 直接通知
        {ID: "e-v-n", WorkflowID: "wf-order", SourceID: "step-validate", TargetID: "step-notify", Label: "rejected"},
        // 支付成功 → 发货
        {ID: "e-p-s", WorkflowID: "wf-order", SourceID: "step-pay", TargetID: "step-ship"},
        // 发货完成 → 通知
        {ID: "e-s-n", WorkflowID: "wf-order", SourceID: "step-ship", TargetID: "step-notify"},
    },
}
```

对应的 DAG 结构：

```
                   ┌─────────┐
                   │  验单    │
                   └────┬────┘
                   /          \
            approved          rejected
              /                  \
        ┌────┴────┐         ┌────┴────┐
        │  支付    │         │  通知    │
        └────┬────┘         └─────────┘
              │
        ┌────┴────┐
        │  发货    │
        └────┬────┘
              │
        ┌────┴────┐
        │  通知    │
        └─────────┘
```

---

## 节点类型

| 类型 | 常量 | 说明 | Config 格式 |
|------|------|------|------------|
| **HTTP** | `NodeTypeHttp` | 调用外部 HTTP 接口 | `{"url":"...","method":"POST","headers":{}}` |
| **SCRIPT** | `NodeTypeScript` | 执行 JS 脚本（goja 引擎，无 CGO） | `{"lang":"js","script":"input.amount * 2"}` |
| **SUB_WORKFLOW** | `NodeTypeSubWorkflow` | ⚠️ 规划中（当前返回未实现错误） | — |
| **DELAY** | `NodeTypeDelay` | 延迟等待 | `{"duration":5}`（秒） |
| **CONDITION** | `NodeTypeCondition` | 条件判断（字段匹配或 JS 表达式） | `{"field":"status","operator":"equals","expected":"active"}` 或 `{"expression":"input.score > 60"}` |
| **CUSTOM** | `NodeTypeCustom` | 自定义执行器 | 自定义 |

> **说明**：
> - `SCRIPT` 支持 `js`/`javascript`，脚本内可用 `input`（解析后的 JSON 对象）与 `inputRaw`（原始字符串）；默认 30s 超时中断防死循环。
> - `SUB_WORKFLOW` 节点当前未实现（executor 返回明确错误，不会静默透传），请将子流程在工作流定义中内联展开或使用自定义执行器。

### HTTP 执行器

```json
{
  "url": "http://api.example.com/action",
  "method": "POST",
  "headers": {
    "Authorization": "Bearer xxx"
  },
  "timeout": 30
}
```

HTTP 执行器会自动从 context 中提取 Trace ID 注入到 `X-Trace-ID` 请求头（复用 `tools/httpclient.WithTrace`）。

### DELAY 执行器

```json
{"duration": 10}
```

延迟 10 秒后继续执行。适用于"等待一段时间后再处理"的场景。

---

## DAG 调度

引擎使用 Kahn BFS 算法进行拓扑排序，DFS 三色标记法做环检测。

### 调度过程

```
1. 加载工作流定义 → 构建 DAG 图
2. 找到入度为 0 的根节点（Root Nodes）
3. 并行执行所有就绪节点
4. 节点完成后，检测其下游节点是否所有上游都已就绪
5. 如果是，加入就绪队列
6. 重复 3-5 直到所有节点执行完毕
```

### 并行控制

通过 `EngineConfig.MaxParallelTasks` 控制并发数（信号量 Semaphore 实现）。

```
节点 A ──► 节点 B ──► 节点 D
                │
节点 C ────────┘

执行顺序：
  T1: A, C (并行，入度为 0)
  T2: B   (A, C 都完成)
  T3: D   (B 完成)
```

### 环检测

如果 DAG 中存在环，引擎在构建时会返回错误：

```go
dag, err := NewDAGGraph(wf)
if err != nil {
    // "workflow DAG has cycle: [nodeA nodeB nodeC nodeA]"
}
```

---

## 分布式调度器

调度器（Scheduler）负责从存储中拉取待执行的任务并分配到 Worker。

### etcd 模式（生产推荐）

复用 `frame/pkg/registry` 通过 etcd 注册 Worker：

```go
import (
    "github.com/LandcLi/landc-go/frame/pkg/registry"
    "github.com/LandcLi/landc-go/workflow/pkg/scheduler"
)

// 创建 etcd 注册中心
etcdReg, _ := registry.NewEtcdRegistry(registry.EtcdRegistryConfig{
    Endpoints: []string{"127.0.0.1:2379"},
})

// 创建调度器
sched := scheduler.NewScheduler(
    dbStore, eng, etcdReg,
    "worker-1", "default", "10.0.0.1:9090",
    scheduler.DefaultSchedulerConfig(),
)

// 启动（自动 etcd 注册 + 续约）
sched.Start(ctx)
defer sched.Stop()
```

### DB 模式（回退）

当 etcd 不可用时，可传入 `nil` 使用 DB 注册回退：

```go
sched := scheduler.NewScheduler(dbStore, eng, nil, "worker-1", "", "", config)
```

---

## 可观测性

### 结构化日志

复用 `log/facade.Logger`，所有工作流事件输出结构化日志：

```go
observer.NewLogObserver(facade.GetLoggerWithName("myapp"))
```

日志示例：

```
2024-01-15T10:30:00.123+0800 INFO [wf] execution started {"execution_id":"xxx","workflow_id":"yyy","trigger_type":"API"}
2024-01-15T10:30:00.456+0800 INFO [wf] task started {"task_id":"t1","node_name":"验单","node_type":"HTTP"}
2024-01-15T10:30:01.000+0800 WARN [wf] task retrying {"task_id":"t1","node_name":"验单","attempt":2,"max_retries":3}
2024-01-15T10:30:02.000+0800 INFO [wf] task completed {"task_id":"t1","node_name":"验单","retry_count":1}
2024-01-15T10:30:05.000+0800 INFO [wf] execution completed {"execution_id":"xxx","status":"COMPLETED"}
```

### 链路追踪

所有 Engine 调用自动创建 `trace.NewSpan`：

```go
// Engine.StartWorkflow 内部会自动创建 span:
ctx, span := trace.NewSpan(ctx, "workflow.StartWorkflow")
defer span.End()
```

每个节点执行也会创建独立的子 span：

```
Trace: workflow.StartWorkflow
  Span: workflow.StartWorkflow
    Span: workflow.node.验单
    Span: workflow.node.支付
    Span: workflow.node.发货
```

### Metrics

`MetricsObserver` 通过 channel 上报事件，可对接 Prometheus：

```go
metricsCh := make(chan observer.MetricEvent, 1000)
obsManager.Register(observer.NewMetricsObserver(metricsCh))

// 在另一个 goroutine 消费
go func() {
    for evt := range metricsCh {
        // 上报到 Prometheus / OpenTelemetry
    }
}()
```

---

## 幂等性

### 执行级别幂等

通过 `TriggerID` 保证：相同 `(WorkflowID, TriggerID)` 不会重复创建 Execution。

```go
// 第一次调用 → 创建执行
execID1, _ := eng.StartWorkflow(ctx, wf.ID, input, model.TriggerTypeApi, "req-001")

// 第二次调用（完全相同参数）→ 返回第一次的执行 ID
execID2, _ := eng.StartWorkflow(ctx, wf.ID, input, model.TriggerTypeApi, "req-001")

assert(execID1 == execID2) // true
```

### 任务级别幂等

每个 Task 创建时带有 `AttemptID`（UUID），用于节点执行端的去重。

### 存储后端

- **开发/测试**：`idempotent.NewMemoryIdempotencyChecker(ttl)` — 内存存储
- **生产**：通过 `frame/pkg/cache.Cache` 自动切换为 Redis 存储

```go
// Redis 模式（需要先初始化 cache）
redisStore := store.NewRedisStore("wf")
idempCheck := idempotent.NewStoreIdempotencyChecker(redisStore, 24*time.Hour)
```

---

## 暂停/恢复与可重入

### 暂停

```go
eng.PauseWorkflow(ctx, execID)
```

- 立即标记 Execution 为 `PAUSED`
- 正在执行的任务会继续完成，但不会启动新的节点
- 保存当前进度到 `Execution.StateData`

### 恢复（可重入）

```go
eng.ResumeWorkflow(ctx, execID)
```

- 标记 Execution 为 `RUNNING`
- 从 DB 加载已有任务列表
- **已完成的节点自动跳过**
- **从中断处继续推进**

### 场景示例

```
步骤1(2s) → 步骤2(3s) → 步骤3(4s) → 步骤4(1s)
                            ↑
                        在步骤3执行时暂停
                           ↓
                        2s后恢复
                           ↓
                        步骤3继续执行(重入)
                           ↓
                        步骤4执行
                           ↓
                        COMPLETED
```

---

## 重试策略

### 线性重试（LINEAR）

```go
{
    MaxRetries: 3,
    RetryDelay: 5,    // 每次等待5秒
    RetryMode:  "LINEAR",
}
// 失败 → 等5s → 重试 → 等5s → 重试 → 等5s → 重试
```

### 指数退避（EXPONENTIAL）

```go
{
    MaxRetries:  3,
    RetryDelay:  2,        // 初始延迟2秒
    RetryMode:   "EXPONENTIAL",
    RetryMaxDelay: 300,    // 最大延迟300秒
}
// 失败 → 等2s → 重试 → 等4s → 重试 → 等8s → 重试
// 每次重试间隔 = baseDelay * 2^(attempt-1) + Jitter
```

### Jitter 抖动

指数退避自动添加随机 Jitter（抖动），防止多个 Worker 同时重试导致惊群效应。

```
第一次重试: wait = 2 * 2^0 + random(0~1)         ≈ 2~3 秒
第二次重试: wait = 2 * 2^1 + random(0~2)         ≈ 4~6 秒
第三次重试: wait = 2 * 2^2 + random(0~4)         ≈ 8~12 秒
```

---

## Key 设计

### 数据表

| 表名 | 说明 | 关键字段 |
|------|------|----------|
| `wf_workflows` | 工作流定义 | id, name, version, status |
| `wf_nodes` | DAG 节点 | id, workflow_id, type, config, retry 策略 |
| `wf_edges` | DAG 边 | id, workflow_id, source_id, target_id, label |
| `wf_executions` | 执行实例 | id, workflow_id, status, trigger_id, current_node_id |
| `wf_tasks` | 任务实例 | id, execution_id, node_id, status, retry_count, attempt_id |
| `wf_workers` | Worker 注册 | id, address, group, heartbeat |

### Redis Key

| Prefix | 用途 | 过期 |
|--------|------|------|
| `wf:lock:*` | 分布式锁 | 可配置（默认30s） |
| `wf:attempt:*` | 幂等去重 | 可配置（默认24h） |
| `wf:worker:*` | Worker 心跳 | 可配置 |
| `wf:exec:*` | 执行缓存 | 可配置 |

---

## 常见问题

### Q: 如何关闭自动建表？

`Store.AutoMigrate()` 在 `workflow.Init()` 中自动调用。如果要手动控制建表，可以不调 `Init()`，而是手动创建 Store 并调 `AutoMigrate()`。

### Q: 如何添加自定义节点类型？

```go
// 1. 实现 NodeExecutor 接口
type MyExecutor struct{}
func (e *MyExecutor) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
    // 自定义逻辑
}
func (e *MyExecutor) Type() string { return "MY_TYPE" }

// 2. 注册到 Registry
execReg.Register(&MyExecutor{})
```

### Q: Engine 的 StartWorkflow 是同步还是异步？

异步。`StartWorkflow` 创建 Execution 记录后立即返回，执行在 goroutine 中异步进行。要等待完成，需要轮询 `GetExecutionStatus`。

### Q: 如何自定义重试策略？

```go
// 实现 RetryStrategy 接口
type CustomRetry struct{}
func (r *CustomRetry) NextDelay(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
    return time.Duration(attempt*attempt) * time.Second
}
func (r *CustomRetry) MaxAttempts() int { return 5 }

// 使用自定义策略
strategy := &CustomRetry{}
retryExec := executor.NewRetryableExecutor(inner, strategy, true)
```

### Q: Scheduler 的 etcd 和 DB 模式有什么区别？

| 特性 | etcd 模式 | DB 模式 |
|------|-----------|---------|
| Worker 注册 | etcd Lease + KeepAlive | DB 定时心跳 |
| Worker 发现 | Watch 实时发现 | 查询 DB |
| 心跳 | etcd TTL 自动 | 手动 Update |
| 适用场景 | 生产环境多 Worker | 开发/单 Worker |

### Q: 如何配置 etcd？

```go
import "github.com/LandcLi/landc-go/frame/pkg/registry"

etcdReg, err := registry.NewEtcdRegistry(registry.EtcdRegistryConfig{
    Endpoints:   []string{"127.0.0.1:2379", "127.0.0.1:2380"},
    DialTimeout: 5 * time.Second,
    Username:    "",
    Password:    "",
    Prefix:      "/services/",
    TTL:         15, // 租约 TTL 秒
})
```

---

## 依赖关系

```
workflow
  ├── frame   → registry, cache, trace, web, config, di, meta, auth
  ├── log     → facade (Logger)
  └── tools   → generate, httpclient
```
