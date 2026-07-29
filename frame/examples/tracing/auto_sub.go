package examples

import (
	"context"
	"fmt"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
	"github.com/LandcLi/landc-go/frame/pkg/trace"
)

var (
	SubProcess = &cmd.Command{
		Name:        "sub_process",
		Brief:       "Sub process with automatic tracing",
		EnableTrace: true,
		Func: func(ctx context.Context, parser *cmd.Parser) error {
			// 上下文会自动从环境变量恢复，无需手动调用
			trace.LogInfo(ctx, "Sub process started (auto-recovered trace context)")
			trace.LogDebug(ctx, "Debug message in sub process")
			trace.LogWarn(ctx, "Warning message in sub process")

			// 显示追踪信息
			info := trace.GetTraceInfo(ctx)
			fmt.Printf("Trace Info: %+v\n", info)

			// 模拟一些处理
			trace.LogInfo(ctx, "Processing data...")

			return nil
		},
	}
)

func RunAutoSub() {
	// 只需要传入 context.Background()，追踪上下文会自动从环境变量恢复
	ctx := context.Background()

	err := SubProcess.Run(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
