package edit

import (
	"context"
	"fmt"
	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/zendev-sh/goai"
	"strings"
)

const Name = "notes_edit"
const Description = "Update the body of an existing persistent note by name."

type Input struct {
	Name string `json:"name" jsonschema:"description=Unique name of the note."`
	Body string `json:"body" jsonschema:"description=Complete replacement note content."`
}
type Output struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) { return execute(ctx, d, in) })
}
func (in Input) Validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(in.Body) == "" {
		return fmt.Errorf("body is required")
	}
	return nil
}
