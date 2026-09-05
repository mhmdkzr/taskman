// Package deno implements Deno command execution.
package deno

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const cmd = "deno"

func execute(ctx context.Context, in input) (tools.Output, error) {
	result, err := tools.ExecuteCMD(ctx, tools.Input{
		Cmd:  cmd,
		Args: in.Args,
	})
	if err != nil {
		return result, fmt.Errorf("deno: %w", err)
	}
	return result, nil
}
