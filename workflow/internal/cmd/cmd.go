package cmd

import (
	"context"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
	"github.com/LandcLi/landc-go/frame/pkg/web"
	api "github.com/LandcLi/landc-go/workflow/api"
)

// Main 工作流服务启动命令
var Main = cmd.NewCommand("workflow", "start workflow HTTP server", func(ctx context.Context, parser *cmd.Parser) error {
	server := web.New()
	if err := server.RegisterHandler(api.GetWorkflowController()); err != nil {
		return err
	}
	if err := server.RegisterHandler(api.GetExecutionController()); err != nil {
		return err
	}
	return server.RunWithContext(ctx)
})
