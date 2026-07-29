package examples

import (
	"context"
	"fmt"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
	"github.com/LandcLi/landc-go/frame/pkg/trace"
)

var (
	Main = &cmd.Command{
		Name:        "main",
		Brief:       "Main command with automatic tracing",
		EnableTrace: true,
		Func: func(ctx context.Context, parser *cmd.Parser) error {
			trace.LogInfo(ctx, "Main process started")

			// 调用子命令，追踪上下文会自动传递
			return cmd.ShellRun(ctx, "go run sub_process.go")
		},
	}
)

func RunAutoMain() {
	ctx := context.Background()
	err := Main.Run(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
