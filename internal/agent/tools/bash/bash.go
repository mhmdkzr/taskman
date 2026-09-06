// Package bash implements shell command execution through bash.
package bash

import (
	"context"
	"errors"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const cmd = "bash"

var ErrEmptyCommand = errors.New("command must not be empty")

func execute(ctx context.Context, in Input) (tools.Output, error) {
	result, err := tools.ExecuteCMD(ctx, tools.Input{
		Cmd:  cmd,
		Args: []string{"-c", in.Command},
	})
	if err != nil {
		return result, fmt.Errorf("bash: %w", err)
	}
	return result, nil
}
