// Package rg implements ripgrep searches.
package rg

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const cmd = "rg"

func execute(ctx context.Context, in input) (tools.Output, error) {
	result, err := tools.ExecuteCMD(ctx, tools.Input{
		Cmd:  cmd,
		Args: in.Args,
	})
	if err != nil {
		return result, fmt.Errorf("rg: %w", err)
	}
	return result, nil
}
