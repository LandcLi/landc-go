package examples

import (
	"context"
	"fmt"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
	"github.com/LandcLi/landc-go/frame/pkg/trace"
)

var (
	Main1 = &cmd.Command{
		Name:        "main",
		Brief:       "Main command with tracing support",
		EnableTrace: true,
		Func: func(ctx context.Context, parser *cmd.Parser) error {
			trace.LogInfo(ctx, "Main process started")

			// 调用子命令
			return cmd.ShellRun(ctx, "go run sub.go")
		},
	}
)

func RunMain() {
	ctx := context.Background()
	err := Main1.Run(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
