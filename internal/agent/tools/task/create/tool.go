// Package create provides the agent task-creation tool.
package create

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
	Name        = "task_create"
	Description = "Create a task for agent execution. New tasks start in the created state."
)

type Input struct {
	Definition    string         `json:"definition" jsonschema:"description=What needs to be done."`
	Specification string         `json:"specification" jsonschema:"description=How the task should be done."`
	Labels        []string       `json:"labels,omitempty" jsonschema:"description=Labels for categorizing the task."`
	Importance    taskrepo.Level `json:"importance" jsonschema:"description=Task importance: 1 (very-low), 2 (low), 3 (medium), 4 (high), or 5 (very-high)."`
	Urgency       taskrepo.Level `json:"urgency" jsonschema:"description=Task urgency: 1 (very-low), 2 (low), 3 (medium), 4 (high), or 5 (very-high)."`
	Complexity    taskrepo.Level `json:"complexity" jsonschema:"description=Task complexity: 1 (very-low), 2 (low), 3 (medium), 4 (high), or 5 (very-high)."`
	Effort        taskrepo.Level `json:"effort" jsonschema:"description=Task effort: 1 (very-low), 2 (low), 3 (medium), 4 (high), or 5 (very-high)."`
	Risk          taskrepo.Level `json:"risk" jsonschema:"description=Task risk: 1 (very-low), 2 (low), 3 (medium), 4 (high), or 5 (very-high)."`
	Autonomy      taskrepo.Level `json:"autonomy" jsonschema:"description=Task autonomy: 1 (very-low), 2 (low), 3 (medium), 4 (high), or 5 (very-high)."`
	Model         string         `json:"model" jsonschema:"description=Model to use for the task."`
	CommitHash    string         `json:"commit_hash,omitempty" jsonschema:"description=Commit hash associated with the task."`
}

type Output struct {
	ID    uuid.UUID          `json:"id"`
	State taskrepo.TaskState `json:"state"`
}

func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, d.Store, in)
	})
}

func (in Input) Validate() error {
	if strings.TrimSpace(in.Definition) == "" {
		return errors.New("definition is required")
	}
	if strings.TrimSpace(in.Specification) == "" {
		return errors.New("specification is required")
	}
	for _, level := range []taskrepo.Level{
		in.Importance, in.Urgency, in.Complexity, in.Effort, in.Risk, in.Autonomy,
	} {
		if level != 0 {
			if err := level.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func execute(ctx context.Context, st *store.Store, in Input) (Output, error) {
	if st == nil {
		return Output{}, errors.New("database is required")
	}

	levelOrDefault := func(level taskrepo.Level) taskrepo.Level {
		if level == 0 {
			return taskrepo.LevelVeryLow
		}
		return level
	}
	id := uuid.NewV7()
	if err := taskrepo.CreateTask(ctx, st.RW(), taskrepo.Task{
		ID:            id,
		Definition:    in.Definition,
		Specification: in.Specification,
		State:         taskrepo.TaskStateCreated,
		Labels:        in.Labels,
		Importance:    levelOrDefault(in.Importance),
		Urgency:       levelOrDefault(in.Urgency),
		Complexity:    levelOrDefault(in.Complexity),
		Effort:        levelOrDefault(in.Effort),
		Risk:          levelOrDefault(in.Risk),
		Autonomy:      levelOrDefault(in.Autonomy),
		Model:         in.Model,
		CommitHash:    in.CommitHash,
	}); err != nil {
		return Output{}, fmt.Errorf("create task: %w", err)
	}
	return Output{ID: id, State: taskrepo.TaskStateCreated}, nil
}
