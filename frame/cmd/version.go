package main

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
)

// version 可通过 -ldflags "-X main.version=x.y.z" 覆盖；
// 默认读取二进制构建信息（go install @version 时即为对应 tag）。
var version = "dev"

// cliVersion 返回 landc 版本：优先构建信息，其次编译期注入。
func cliVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return version
}

func NewVersionCommand() *cmd.Command {
	return cmd.NewCommand("version", "Show landc version information", func(ctx context.Context, parser *cmd.Parser) error {
		fmt.Printf("landc %s\n", cliVersion())
		fmt.Printf("go %s\n", runtime.Version())
		fmt.Printf("os/arch %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return nil
	})
}
