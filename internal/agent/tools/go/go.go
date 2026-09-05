// Package gocmd implements Go command execution.
package gocmd

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const cmd = "go"

func execute(ctx context.Context, in input) (tools.Output, error) {
	result, err := tools.ExecuteCMD(ctx, tools.Input{
		Cmd:  cmd,
		Args: in.Args,
	})
	if err != nil {
		return result, fmt.Errorf("go: %w", err)
	}
	return result, nil
}
