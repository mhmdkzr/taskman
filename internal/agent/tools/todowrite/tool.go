package todowrite

import (
	"context"
	"errors"
	"fmt"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "todowrite"
	description = "Replace the current session's task list with an updated set of todos."
)

const Description = description

type input struct {
	Todos []sessions.Todo `json:"todos" jsonschema:"description=The complete current todo list. Use an empty list when no work remains."`
}

type output struct {
	Todos []sessions.Todo `json:"todos"`
}

type Input = input
type Output = output

func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, description, func(ctx context.Context, in input) (output, error) {
		if d.DB == nil {
			return output{}, errors.New("todowrite: database is required")
		}
		if d.SessionID == (sessions.SessionID{}) {
			return output{}, errors.New("todowrite: session is required")
		}
		if err := sessions.ReplaceTodos(ctx, d.DB, d.SessionID, in.Todos); err != nil {
			return output{}, fmt.Errorf("todowrite: %w", err)
		}
		return output{Todos: in.Todos}, nil
	})
}

func (input) Validate() error { return nil }
