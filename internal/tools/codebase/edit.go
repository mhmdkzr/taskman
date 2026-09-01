package codebase

import (
	"context"
	"fmt"

	cb "github.com/mhmdkzr/taskman/internal/codebase"
	"github.com/zendev-sh/goai"
)

type editInput struct {
	Path string `json:"path" jsonschema:"description=Path of the file, relative to the repository root."`
	Old  string `json:"old,omitempty" jsonschema:"description=Exact text to replace. Leave empty to create the file (if it does not exist) or append (if it does)."`
	New  string `json:"new" jsonschema:"description=Replacement text, new file content, or text to append, depending on old and whether the file exists."`
}

// EditTool returns the edit tool bound to repo: create, append, or
// exact-match replace, chosen by whether old is set and whether the file
// already exists (see Repository.Edit).
func EditTool(repo cb.Repository) goai.Tool {
	return goai.NewTool("edit",
		"Create, append to, or replace text in a file. Leave old empty to create a new file or append to an existing one; "+
			"set old to the exact text to replace (it must match exactly once, otherwise this errors).",
		func(ctx context.Context, in editInput) (string, error) {
			res, err := repo.Edit(in.Path, in.Old, in.New)
			if err != nil {
				return "", err
			}
			if res.Created {
				return fmt.Sprintf("created %s (%d bytes)", in.Path, res.BytesWritten), nil
			}
			if in.Old == "" {
				return fmt.Sprintf("appended %d bytes to %s", res.BytesWritten, in.Path), nil
			}
			return fmt.Sprintf("edited %s", in.Path), nil
		})
}
