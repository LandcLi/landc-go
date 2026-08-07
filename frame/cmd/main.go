package main

import (
	"context"
	"fmt"
	"os"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
)

func main() {
	app := cmd.NewApp()
	app.Name = "landc"
	app.Brief = "landc-go CLI tool for project management"
	app.Description = "A command-line tool for managing landc-go projects, including initialization, code generation, and more."
	// CLI 为离线开发工具：跳过自动 bootstrap（config/db/cache 初始化），避免无谓连接与延迟。
	app.Bootstrap = nil

	if err := app.AddCommand(
		NewInitCommand(),
		NewGenCommand(),
		NewMigrateDBContextCommand(),
		NewVersionCommand(),
		NewDoctorCommand(),
	); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := app.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
