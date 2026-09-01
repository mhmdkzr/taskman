package codebase

import (
	"context"
	"fmt"
	"strings"

	"github.com/zendev-sh/goai"

	cb "github.com/mhmdkzr/taskman/internal/codebase"
)

type grepInput struct {
	Pattern string  `json:"pattern"           jsonschema:"description=Regular expression to search for."`
	Path    *string `json:"path,omitempty"    jsonschema:"description=File or directory to search, relative to the repository root (default .)."`
	Include *string `json:"include,omitempty" jsonschema:"description=Glob pattern (supports **) that matching file paths must satisfy."`
	Limit   *int    `json:"limit,omitempty"   jsonschema:"description=Maximum number of matches to return (default 100)."`
}

// GrepTool returns the grep tool bound to repo.
func GrepTool(repo cb.Repository) goai.Tool {
	return goai.NewTool(
		"grep",
		"Search files for lines matching a regular expression. Returns file:line:content matches. Searches a directory (default .) or a single file.",
		func(ctx context.Context, in grepInput) (string, error) {
			path := ""
			if in.Path != nil {
				path = *in.Path
			}
			include := ""
			if in.Include != nil {
				include = *in.Include
			}
			limit := 0
			if in.Limit != nil {
				limit = *in.Limit
			}
			matches, truncated, err := repo.Grep(in.Pattern, path, include, limit)
			if err != nil {
				return "", err
			}
			if len(matches) == 0 {
				return "no matches", nil
			}
			lines := make([]string, len(matches))
			for i, m := range matches {
				lines[i] = fmt.Sprintf("%s:%d:%s", m.Path, m.Line, m.Text)
			}
			out := strings.Join(lines, "\n")
			if truncated {
				out += fmt.Sprintf("\n... (truncated; %d matches shown)", len(matches))
			}
			return out, nil
		},
	)
}
