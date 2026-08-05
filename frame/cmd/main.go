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

	if err := app.AddCommand(
		NewInitCommand(),
		NewGenCommand(),
		NewMigrateDBContextCommand(),
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
