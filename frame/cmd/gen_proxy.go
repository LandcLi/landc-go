package main

import (
	"context"
	"fmt"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
	"github.com/LandcLi/landc-go/frame/pkg/proxygen"
)

// NewGenCommand creates the "gen" parent command.
func NewGenCommand() *cmd.Command {
	genCmd := cmd.NewCommand("gen", "Code generation tools", nil)

	_ = genCmd.AddCommand(
		NewGenProxyCommand(),
	)

	return genCmd
}

// NewGenProxyCommand creates the "gen proxy" command.
func NewGenProxyCommand() *cmd.Command {
	proxyCmd := cmd.NewCommand("proxy", "Generate HTTP proxy code for a controller interface", func(ctx context.Context, parser *cmd.Parser) error {
		interfaceName := parser.GetOpt("type")
		gatewayName := parser.GetOpt("gateway-name")

		if interfaceName == "" {
			return fmt.Errorf("flag -type is required (e.g. -type UserController)")
		}
		if gatewayName == "" {
			// Default: lowercase interface name + ".controller"
			gatewayName = string(interfaceName[0]-'A'+'a') + interfaceName[1:] + ".controller"
		}

		dir := parser.GetOpt("dir")
		output := parser.GetOpt("output")

		cfg := proxygen.Config{
			InterfaceName: interfaceName,
			GatewayName:   gatewayName,
			Dir:           dir,
			Output:        output,
		}

		return proxygen.Generate(cfg)
	})

	proxyCmd.AddOption("type", true)          // -type <interface name>
	proxyCmd.AddOption("gateway-name", true)  // -gateway-name <gateway name>
	proxyCmd.AddOption("dir", true)           // -dir <package directory>
	proxyCmd.AddOption("output", true)        // -output <output file>

	return proxyCmd
}
