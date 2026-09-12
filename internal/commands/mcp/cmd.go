// Package mcp wires the "mcp" command: it serves the same task operations
// as `taskman <command>`, over MCP/stdio instead of flags and stdout.
package mcp

import (
	"context"
	"fmt"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/mcp"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "mcp" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "mcp",
		Usage: "serve task operations over MCP/stdio instead of the CLI",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := utils.StoreFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			defer st.Close()

			server := mcp.NewServer(st, utils.GitFrom(cmd))
			if err := server.Run(ctx, &gosdkmcp.StdioTransport{}); err != nil {
				return fmt.Errorf("serve mcp: %w", err)
			}
			return nil
		},
	}
}
