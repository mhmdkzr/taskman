// Package update provides the agent task update tool.
package update

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"uuid"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
	taskrepo "github.com/mhmdkzr/loop/internal/agent/tools/task"
	"github.com/mhmdkzr/loop/internal/store"
)

const (
	Name        = "task_update"
	Description = "Update an existing task and replace its labels."
)

type Input struct {
	ID              string             `json:"id"                         jsonschema:"description=Task UUID."`
	Definition      string             `json:"definition"                 jsonschema:"description=What needs to be done."`
	Specification   string             `json:"specification"              jsonschema:"description=How the task should be done."`
	State           taskrepo.TaskState `json:"state"                      jsonschema:"description=Task state."`
	Labels          []string           `json:"labels,omitempty"           jsonschema:"description=Labels for categorizing the task."`
	Importance      taskrepo.Level     `json:"importance"`
	Urgency         taskrepo.Level     `json:"urgency"`
	Complexity      taskrepo.Level     `json:"complexity"`
	Effort          taskrepo.Level     `json:"effort"`
	Risk            taskrepo.Level     `json:"risk"`
	Autonomy        taskrepo.Level     `json:"autonomy"`
	Model           string             `json:"model"`
	ReasoningEffort string             `json:"reasoning_effort,omitempty"`
	CommitHash      string             `json:"commit_hash,omitempty"`
	Branch          string             `json:"branch,omitempty"`
	FailureReason   string             `json:"failure_reason,omitempty" jsonschema:"description=Why the task failed (e.g. rejected review feedback or a processing error)."`
}

type Output struct {
	Task taskrepo.Task `json:"task"`
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
	if strings.TrimSpace(in.Definition) == "" {
		return errors.New("definition is required")
	}
	if strings.TrimSpace(in.Specification) == "" {
		return errors.New("specification is required")
	}
	if !validState(in.State) {
		return fmt.Errorf("invalid state: %q", in.State)
	}
	if strings.TrimSpace(in.Model) == "" {
		return errors.New("model is required")
	}
	for _, level := range []taskrepo.Level{
		in.Importance, in.Urgency, in.Complexity, in.Effort, in.Risk, in.Autonomy,
	} {
		if err := level.Validate(); err != nil {
			return err
		}
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
	// Model is a free-text name (task.Task.Model, not a models.model_id FK),
	// so it isn't caught by any DB constraint - resolve it now rather than
	// leaving a typo to surface only once something tries to dispatch a
	// session against it.
	if _, err := sessions.ModelIDByName(ctx, st.RO(), in.Model); err != nil {
		return Output{}, fmt.Errorf("resolve model %q: %w", in.Model, err)
	}
	t := taskrepo.Task{
		ID: id, Definition: in.Definition, Specification: in.Specification, State: in.State,
		Labels: in.Labels, Importance: in.Importance, Urgency: in.Urgency,
		Complexity: in.Complexity, Effort: in.Effort, Risk: in.Risk, Autonomy: in.Autonomy,
		Model: in.Model, ReasoningEffort: in.ReasoningEffort, CommitHash: in.CommitHash, Branch: in.Branch,
		FailureReason: in.FailureReason,
	}
	if err := taskrepo.UpdateTask(ctx, st.RW(), t); err != nil {
		return Output{}, fmt.Errorf("update task: %w", err)
	}
	return Output{Task: t}, nil
}

func validState(state taskrepo.TaskState) bool {
	switch state {
	case taskrepo.TaskStateCreated, taskrepo.TaskStateStarted, taskrepo.TaskStateCompleted,
		taskrepo.TaskStateCancelled, taskrepo.TaskStateBlocked, taskrepo.TaskStateFailed:
		return true
	default:
		return false
	}
}
