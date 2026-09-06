package link

import (
	"context"
	"fmt"
	"strings"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "notes_link"
	Description = "Create or update a link between two existing notes by name."
)

type Input struct {
	From         string `json:"from"                   jsonschema:"description=First note name."`
	To           string `json:"to"                     jsonschema:"description=Name of the second note."`
	Relationship string `json:"relationship,omitempty" jsonschema:"description=Optional relationship description."`
}
type Output struct {
	From         string `json:"from"`
	To           string `json:"to"`
	Relationship string `json:"relationship"`
}

func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, d, in)
	})
}

func (in Input) Validate() error {
	a, b := strings.TrimSpace(in.From), strings.TrimSpace(in.To)
	if a == "" || b == "" {
		return fmt.Errorf("from and to are required")
	}
	if a == b {
		return fmt.Errorf("a note cannot link to itself")
	}
	return nil
}
