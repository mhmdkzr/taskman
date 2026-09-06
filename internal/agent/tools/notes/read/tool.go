package read

import (
	"context"
	"fmt"
	"strings"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "notes_read"
	Description = "Read one persistent note by name, including its links."
)

type Input struct {
	Name string `json:"name" jsonschema:"description=Unique name of the note."`
}
type Link struct {
	Name         string `json:"name"`
	Relationship string `json:"relationship"`
}
type Output struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	Links     []Link `json:"links"`
}

func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, d, in)
	})
}

func (in Input) Validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
