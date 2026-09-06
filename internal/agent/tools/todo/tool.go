package todo

import (
	"context"
	"errors"
	"fmt"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "todo"
	Description = "Replace the current session's task list with an updated set of todos."
)

type Input struct {
	Todos []sessions.Todo `json:"todos" jsonschema:"description=The complete current todo list. Use an empty list when no work remains."`
}

type Output struct {
	Todos []sessions.Todo `json:"todos"`
}

func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		if d.Store == nil {
			return Output{}, errors.New("todo: database is required")
		}
		if d.SessionID == (sessions.SessionID{}) {
			return Output{}, errors.New("todo: session is required")
		}
		if err := sessions.ReplaceTodos(ctx, d.Store, d.SessionID, in.Todos); err != nil {
			return Output{}, fmt.Errorf("todo: %w", err)
		}
		return Output{Todos: in.Todos}, nil
	})
}

func (Input) Validate() error { return nil }
