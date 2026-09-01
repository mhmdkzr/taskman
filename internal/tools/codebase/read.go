package codebase

import (
	"context"
	"fmt"
	"strings"

	"github.com/zendev-sh/goai"

	cb "github.com/mhmdkzr/taskman/internal/codebase"
)

type readInput struct {
	Path   string `json:"path"             jsonschema:"description=Path to the file to read, relative to the repository root."`
	Offset *int   `json:"offset,omitempty" jsonschema:"description=0-based line number to start from (default 0)."`
	Limit  *int   `json:"limit,omitempty"  jsonschema:"description=Maximum number of lines to return (default 2000)."`
}

// ReadTool returns the read tool bound to repo.
func ReadTool(repo cb.Repository) goai.Tool {
	return goai.NewTool(
		"read",
		"Read a file and return its content line by line. Use offset and limit to page through large files instead of loading them whole.",
		func(ctx context.Context, in readInput) (string, error) {
			offset := 0
			if in.Offset != nil {
				offset = *in.Offset
			}
			limit := 0
			if in.Limit != nil {
				limit = *in.Limit
			}
			lines, truncated, err := repo.Read(in.Path, offset, limit)
			if err != nil {
				return "", err
			}
			out := strings.Join(lines, "\n")
			if truncated {
				out += fmt.Sprintf("\n... (truncated; continue reading with offset=%d)", offset+len(lines))
			}
			return out, nil
		},
	)
}
