package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

type Input struct {
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
}

type Output struct {
	StdOut   string `json:"stdout"`
	StdErr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exit_code"`
}

func ExecuteCMD(ctx context.Context, in Input) (Output, error) {
	//nolint:gosec // This tool intentionally executes the command supplied by the agent.
	command := exec.CommandContext(ctx, in.Cmd, in.Args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr

	stdout, err := command.Output()
	result := Output{
		StdOut: string(stdout),
		StdErr: stderr.String(),
	}
	if command.ProcessState != nil {
		result.ExitCode = command.ProcessState.ExitCode()
	}
	if err != nil {
		return result, fmt.Errorf("execute %q: %w", in.Cmd, err)
	}
	return result, nil
}
