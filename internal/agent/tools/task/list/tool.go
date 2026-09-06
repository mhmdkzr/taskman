// Package list provides the agent task listing tool.
package list

import (
	"context"
	"errors"
	"fmt"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	taskrepo "github.com/mhmdkzr/loop/internal/agent/tools/task"
	"github.com/mhmdkzr/loop/internal/store"
)

const (
	Name        = "task_list"
	Description = "List tasks using optional state, label, planning, model, commit, or ID filters."
)

type Input struct {
	Filter taskrepo.TaskFilter `json:"filter,omitempty" jsonschema:"description=Optional task filters."`
}

type Output struct {
	Tasks []taskrepo.Task `json:"tasks"`
}

func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, d.Store, in)
	})
}

func (in Input) Validate() error {
	for name, levels := range map[string][]taskrepo.Level{
		"importance": in.Filter.Importance,
		"urgency":    in.Filter.Urgency,
		"complexity": in.Filter.Complexity,
		"effort":     in.Filter.Effort,
		"risk":       in.Filter.Risk,
		"autonomy":   in.Filter.Autonomy,
	} {
		for _, level := range levels {
			if err := level.Validate(); err != nil {
				return fmt.Errorf("invalid %s: %d: %w", name, level, err)
			}
		}
	}
	return nil
}

func execute(ctx context.Context, st *store.Store, in Input) (Output, error) {
	if st == nil {
		return Output{}, errors.New("database is required")
	}
	tasks, err := taskrepo.ListTasks(ctx, st.RO(), in.Filter)
	if err != nil {
		return Output{}, fmt.Errorf("list tasks: %w", err)
	}
	return Output{Tasks: tasks}, nil
}
