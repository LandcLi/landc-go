package main

import (
	"context"
	"fmt"

	"github.com/LandcLi/landc-go/mvc/pkg/cmd"
	"github.com/LandcLi/landc-go/mvc/pkg/trace"
)

var (
	Sub = &cmd.Command{
		Name:        "sub",
		Brief:       "Sub command with tracing support",
		EnableTrace: true,
		Func: func(ctx context.Context, parser *cmd.Parser) error {
			trace.LogInfo(ctx, "Sub process started")
			trace.LogDebug(ctx, "Debug message in sub process")
			trace.LogWarn(ctx, "Warning message in sub process")

			// 模拟一些处理
			trace.LogInfo(ctx, "Processing data...")

			return nil
		},
	}
)

func RunSub() {
	ctx := context.Background()

	err := Sub.Run(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
