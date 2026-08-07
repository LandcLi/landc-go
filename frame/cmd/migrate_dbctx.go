package main

import (
	"context"
	"fmt"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
	"github.com/LandcLi/landc-go/frame/pkg/codemod"
)

// NewMigrateDBContextCommand 创建 migrate-dbctx 命令。
//
// 用法：landc migrate-dbctx [--dry-run] <dir> [dir ...]
// 为目录下 DAO/service 接口与方法注入 ctx 参数，并把
// db.GetDB() / cache.GetCache() / context.Background() 迁移为
// db.GetDBFrom(ctx) / cache.GetCacheFrom(ctx) / ctx。
// --dry-run 仅预览将修改的文件，不写盘。
func NewMigrateDBContextCommand() *cmd.Command {
	command := cmd.NewCommand(
		"migrate-dbctx",
		"Migrate DAO/service to context-aware resource access (ctx + GetDBFrom/GetCacheFrom)",
		func(ctx context.Context, parser *cmd.Parser) error {
			dirs := parser.GetArgAll()
			if len(dirs) == 0 {
				return fmt.Errorf("usage: landc migrate-dbctx [--dry-run] <dir> [dir ...]")
			}
			dryRun := parser.HasOpt("dry-run")
			verb := "migrated"
			if dryRun {
				verb = "would migrate (dry-run, nothing written)"
			}
			for _, dir := range dirs {
				var modified []string
				var err error
				if dryRun {
					modified, err = codemod.MigrateDBContextDryRun(dir)
				} else {
					modified, err = codemod.MigrateDBContext(dir)
				}
				if err != nil {
					return fmt.Errorf("migrate %s: %w", dir, err)
				}
				fmt.Printf("[%s] %s %d file(s)\n", dir, verb, len(modified))
				for _, f := range modified {
					fmt.Printf("  - %s\n", f)
				}
				if !dryRun {
					fmt.Println("  -> 调用点（service 调 dao 等）请按 go build 错误列表逐个补 ctx 参数")
				}
			}
			return nil
		},
	)
	command.AddOption("dry-run", false)
	return command
}
