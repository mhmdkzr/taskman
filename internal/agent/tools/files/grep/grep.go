// Package grep implements structured regular-expression searches.
package grep

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

var errPatternRequired = errors.New("pattern is required")

func execute(ctx context.Context, in input) (output, error) {
	limit := in.MaxResults
	if limit <= 0 {
		limit = 100
	}
	path := in.Path
	if path == "" {
		path = "."
	}

	args := []string{"--line-number", "--color", "never", "--no-heading", "--hidden", "--glob", "!.git"}
	if in.Include != "" {
		args = append(args, "--glob", in.Include)
	}
	args = append(args, "--", in.Pattern, path)
	result, err := tools.ExecuteCMD(ctx, tools.Input{Cmd: "rg", Args: args})
	if err != nil && result.ExitCode != 1 {
		return output{}, fmt.Errorf("grep: %w", err)
	}

	lines := strings.Split(strings.TrimSuffix(result.StdOut, "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	truncated := len(lines) > limit
	if truncated {
		lines = lines[:limit]
	}
	return output{Matches: lines, Truncated: truncated}, nil
}
