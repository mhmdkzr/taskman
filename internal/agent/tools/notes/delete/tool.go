package delete

import (
	"context"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "notes_delete"
	Description = "Delete a persistent note by name, including its links."
)

type Input struct {
	Name string `json:"name" jsonschema:"description=Unique name of the note to delete."`
}
type Output struct {
	Deleted bool   `json:"deleted"`
	Name    string `json:"name"`
}

func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, d, in)
	})
}
func (Input) Validate() error { return nil }
