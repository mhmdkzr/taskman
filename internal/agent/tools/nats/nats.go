// Package nats implements the NATS CLI agent tool.
package nats

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const cmd = "nats"

func execute(ctx context.Context, in Input) (tools.Output, error) {
	result, err := tools.ExecuteCMD(ctx, tools.Input{
		Cmd:  cmd,
		Args: in.Args,
	})
	if err != nil {
		return result, fmt.Errorf("nats: %w", err)
	}
	return result, nil
}
