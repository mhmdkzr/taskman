package codebase

import (
	"context"
	"fmt"
	"os"
	"strings"

	cb "github.com/mhmdkzr/taskman/internal/codebase"
	"github.com/zendev-sh/goai"
)

// GoBuildTool returns the go_build tool bound to repo. The build's output
// binary is always discarded (os.DevNull) — this tool is a compile check,
// not a way to produce an artifact.
func GoBuildTool(repo cb.Repository) goai.Tool {
	return goai.NewTool("go_build",
		"Compile every package in the repository (`go build ./...`) and report any build failures.",
		func(ctx context.Context, in struct{}) (string, error) {
			events, err := repo.GoBuild(os.DevNull)
			if err != nil {
				return "", fmt.Errorf("go_build: %w", err)
			}
			var failures []string
			for _, e := range events {
				if e.Failed() {
					failures = append(failures, fmt.Sprintf("%s:\n%s", e.ImportPath, e.Output))
				}
			}
			if len(failures) == 0 {
				return "build ok", nil
			}
			return "build failed:\n\n" + strings.Join(failures, "\n"), nil
		})
}
