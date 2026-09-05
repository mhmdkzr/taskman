// Package psql implements PostgreSQL CLI command execution.
package psql

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const cmd = "psql"

func execute(ctx context.Context, in input) (tools.Output, error) {
	result, err := tools.ExecuteCMD(ctx, tools.Input{
		Cmd:  cmd,
		Args: in.Args,
	})
	if err != nil {
		return result, fmt.Errorf("psql: %w", err)
	}
	return result, nil
}
