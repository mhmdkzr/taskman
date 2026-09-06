// Package delete provides the agent task deletion tool.
package delete

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"uuid"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	taskrepo "github.com/mhmdkzr/loop/internal/agent/tools/task"
	"github.com/mhmdkzr/loop/internal/store"
)

const (
	Name        = "task_delete"
	Description = "Delete a task by ID."
)

type Input struct {
	ID string `json:"id" jsonschema:"description=Task UUID."`
}

type Output struct {
	ID      uuid.UUID `json:"id"`
	Deleted bool      `json:"deleted"`
}

func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, d.Store, in)
	})
}

func (in Input) Validate() error {
	if strings.TrimSpace(in.ID) == "" {
		return errors.New("id is required")
	}
	if _, err := uuid.Parse(in.ID); err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	return nil
}

func execute(ctx context.Context, st *store.Store, in Input) (Output, error) {
	if st == nil {
		return Output{}, errors.New("database is required")
	}
	id, err := uuid.Parse(in.ID)
	if err != nil {
		return Output{}, fmt.Errorf("invalid id: %w", err)
	}
	if err := taskrepo.DeleteTask(ctx, st.RW(), id); err != nil {
		return Output{}, fmt.Errorf("delete task: %w", err)
	}
	return Output{ID: id, Deleted: true}, nil
}
