package main

import (
	"context"

	_ "github.com/LandcLi/landc-go/examples/demo/internal"
	"github.com/LandcLi/landc-go/examples/demo/internal/cmd"
)

func main() {
	// cmd.Main.Run 自动完成：加载 config.yaml → 初始化 DB/Cache/JWT → 信号监听 + 优雅停机
	cmd.Main.Run(context.Background())
}
