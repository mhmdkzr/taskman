package codebase

import (
	"context"
	"strings"

	cb "github.com/mhmdkzr/taskman/internal/codebase"
	"github.com/zendev-sh/goai"
)

type globInput struct {
	Pattern string  `json:"pattern" jsonschema:"description=Glob pattern to match against paths relative to the root."`
	Path    *string `json:"path,omitempty" jsonschema:"description=Root directory to search, relative to the repository root (default .)."`
}

// GlobTool returns the glob tool bound to repo.
func GlobTool(repo cb.Repository) goai.Tool {
	return goai.NewTool("glob",
		"Find files by name pattern. ** matches across directories, * matches within a directory, ? matches a single character. Hidden files and directories are skipped.",
		func(ctx context.Context, in globInput) (string, error) {
			path := ""
			if in.Path != nil {
				path = *in.Path
			}
			matches, err := repo.Glob(in.Pattern, path)
			if err != nil {
				return "", err
			}
			if len(matches) == 0 {
				return "no matches", nil
			}
			return strings.Join(matches, "\n"), nil
		})
}
