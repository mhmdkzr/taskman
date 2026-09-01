package codebase

import (
	"context"
	"fmt"
	"strings"
	"time"

	cb "github.com/mhmdkzr/taskman/internal/codebase"
	"github.com/zendev-sh/goai"
)

// maxTestOutputChars bounds go_test's returned output so a runaway or very
// verbose test suite cannot flood the agent's context window.
const maxTestOutputChars = 20000

type testInput struct {
	Run     *string `json:"run,omitempty" jsonschema:"description=Regexp matching test names to run (-run). Omit to run everything."`
	Timeout *int    `json:"timeout_seconds,omitempty" jsonschema:"description=Test binary timeout in seconds (default the go tool's own default, 10 minutes)."`
}

// GoTestTool returns the go_test tool bound to repo.
func GoTestTool(repo cb.Repository) goai.Tool {
	return goai.NewTool("go_test",
		"Run `go test -v ./...` (optionally filtered by test name) and return the output plus a pass/fail summary.",
		func(ctx context.Context, in testInput) (string, error) {
			args := cb.GoTestArgs{Verbose: true}
			if in.Run != nil {
				args.Run = *in.Run
			}
			if in.Timeout != nil {
				args.Timeout = time.Duration(*in.Timeout) * time.Second
			}
			events, err := repo.GoTest(args)
			if err != nil {
				return "", fmt.Errorf("go_test: %w", err)
			}
			return formatTestEvents(events), nil
		})
}

func formatTestEvents(events []cb.TestEvent) string {
	var out strings.Builder
	passed, failed := 0, 0
	for _, e := range events {
		if e.Action == cb.TestActionOutput {
			out.WriteString(e.Output)
		}
		if e.IsPackageLevel() {
			switch e.Action {
			case cb.TestActionPass:
				passed++
			case cb.TestActionFail:
				failed++
			default:
				// start/pause/cont/bench/skip don't affect the summary.
			}
		}
	}

	text := out.String()
	truncated := false
	if len(text) > maxTestOutputChars {
		text = text[:maxTestOutputChars]
		truncated = true
	}
	summary := fmt.Sprintf("\n%d package(s) passed, %d failed", passed, failed)
	if truncated {
		summary += fmt.Sprintf(" (output truncated to %d chars)", maxTestOutputChars)
	}
	return text + summary
}
