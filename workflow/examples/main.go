// Example 演示了如何使用 workflow 框架的 Init() + DI 模式。
//
// 运行方式：
//
//	cd examples
//	go run main.go
//
// 前置条件：
//   - 需要有效的 config.yaml（包含 database 配置）
//   - 或使用 memory DB + 手动 InitWithComponents
package main

import (
	"context"
	"fmt"
	"log"

	workflow "github.com/LandcLi/landc-go/workflow"
	_ "github.com/LandcLi/landc-go/workflow/internal" // 触发所有 init() 注册
	"github.com/LandcLi/landc-go/workflow/service"
)

func main() {
	// 方式一：自动从框架配置初始化
	// 需要先调用 frameconfig.InitGlobalConfigWithPath("config.yaml")
	// if err := workflow.Init(); err != nil { ... }

	// 方式二：这里演示直接使用 DI 模式的伪代码思路
	// 实际使用时请替换为 Init() + config.yaml
	_ = workflow.Init
	_ = service.GetWorkflowService
	_ = service.GetExecutionService

	fmt.Println("workflow 框架已就绪")
	fmt.Println("API 端点:")
	fmt.Println("  POST   /api/workflows                    - 创建工作流")
	fmt.Println("  GET    /api/workflows                    - 工作流列表")
	fmt.Println("  GET    /api/workflows/:id                - 查询工作流")
	fmt.Println("  DELETE /api/workflows/:id                - 删除工作流")
	fmt.Println("  POST   /api/workflows/:id/start          - 启动工作流")
	fmt.Println("  GET    /api/executions                   - 执行列表")
	fmt.Println("  GET    /api/executions/:id               - 查询执行")
	fmt.Println("  POST   /api/executions/:id/pause         - 暂停执行")
	fmt.Println("  POST   /api/executions/:id/resume        - 恢复执行")
	fmt.Println("  POST   /api/executions/:id/cancel        - 取消执行")
	fmt.Println("  GET    /api/executions/:id/tasks         - 任务列表")

	// 启动 HTTP 服务（需正确配置 config.yaml）
	// cmd.Main.Run(context.Background())
	_ = context.Background()
	log.Println("examples: 请参考 README.md 配置后运行")
}
