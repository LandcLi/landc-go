package main

import (
	"context"
	"fmt"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
	"github.com/LandcLi/landc-go/frame/pkg/codemod"
)

// NewMigrateDBContextCommand 创建 migrate-dbctx 命令。
//
// 用法：landc migrate-dbctx <dir> [dir ...]
// 为目录下 DAO/service 接口与方法注入 ctx 参数，并把
// db.GetDB() / cache.GetCache() / context.Background() 迁移为
// db.GetDBFrom(ctx) / cache.GetCacheFrom(ctx) / ctx。
func NewMigrateDBContextCommand() *cmd.Command {
	return cmd.NewCommand(
		"migrate-dbctx",
		"Migrate DAO/service to context-aware resource access (ctx + GetDBFrom/GetCacheFrom)",
		func(ctx context.Context, parser *cmd.Parser) error {
			dirs := parser.GetArgAll()
			if len(dirs) == 0 {
				return fmt.Errorf("usage: landc migrate-dbctx <dir> [dir ...]")
			}
			for _, dir := range dirs {
				modified, err := codemod.MigrateDBContext(dir)
				if err != nil {
					return fmt.Errorf("migrate %s: %w", dir, err)
				}
				fmt.Printf("[%s] migrated %d file(s)\n", dir, len(modified))
				for _, f := range modified {
					fmt.Printf("  - %s\n", f)
				}
				fmt.Println("  -> 调用点（service 调 dao 等）请按 go build 错误列表逐个补 ctx 参数")
			}
			return nil
		},
	)
}
