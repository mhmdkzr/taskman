package task

import (
	"context"
	"database/sql"
	"fmt"
	"uuid"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/taskman/internal/task"
)

type editInput struct {
	ID            string   `json:"id"                     jsonschema:"description=Task ID (UUID) to edit. Only works while the task is still unstarted."`
	Title         string   `json:"title"                  jsonschema:"description=Short title (a few words), not a sentence."`
	TaskType      string   `json:"task_type"              jsonschema:"description=One of: bug, docs, feature, refactor, test."`
	Urgency       string   `json:"urgency"                jsonschema:"description=Time pressure: low, medium, or high."`
	Importance    string   `json:"importance"             jsonschema:"description=Impact if never done: low, medium, or high."`
	Risk          string   `json:"risk"                   jsonschema:"description=Blast radius if execution goes wrong: low, medium, or high."`
	What          string   `json:"what"                   jsonschema:"description=What the task is."`
	Why           string   `json:"why"                    jsonschema:"description=Why it matters."`
	How           string   `json:"how"                    jsonschema:"description=How to do it."`
	Where         []string `json:"where,omitempty"        jsonschema:"description=file:line citations."`
	Invariants    []string `json:"invariants,omitempty"   jsonschema:"description=Things that must stay true after the change."`
	Packages      []string `json:"packages"               jsonschema:"description=Packages this task touches."`
	CompletedWhen []string `json:"completed_when"         jsonschema:"description=Independently checkable completion conditions."`
	Tags          []string `json:"tags,omitempty"`
	Dependencies  []string `json:"dependencies,omitempty" jsonschema:"description=IDs of other tasks this one depends on."`
}

// EditTool returns the task_edit tool bound to db. It replaces a task's full
// descriptive spec — the caller must pass every field it wants to keep, not
// just the ones it wants to change (task_get first, then re-submit the
// changed copy, is the intended flow).
func EditTool(db *sql.DB) goai.Tool {
	return goai.NewTool("task_edit",
		"Replace a task's descriptive spec (title, type, urgency/importance/risk, what/why/how, packages, "+
			"completed_when, etc). Only works while the task hasn't started yet — read it with task_get first "+
			"and submit the full updated spec, not a partial patch.",
		func(ctx context.Context, in editInput) (string, error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return "", fmt.Errorf("invalid id %q: %w", in.ID, err)
			}
			if err := validateSpec(in.TaskType, in.Urgency, in.Importance, in.Risk,
				in.Title, in.What, in.Why, in.How, in.Packages, in.CompletedWhen); err != nil {
				return "", err
			}
			existing, err := task.GetTask(ctx, db, id)
			if err != nil {
				return "", fmt.Errorf("task_edit: %w", err)
			}

			existing.Title = in.Title
			existing.TaskType = in.TaskType
			existing.Urgency = in.Urgency
			existing.Importance = in.Importance
			existing.Risk = in.Risk
			existing.What = in.What
			existing.Why = in.Why
			existing.How = in.How
			existing.Where = in.Where
			existing.Invariants = in.Invariants
			existing.Packages = in.Packages
			existing.CompletedWhen = in.CompletedWhen
			existing.Tags = in.Tags
			existing.Dependencies = in.Dependencies

			if err := task.EditTask(ctx, db, *existing); err != nil {
				return "", fmt.Errorf("task_edit: %w", err)
			}
			return fmt.Sprintf("updated task %s: %s", existing.ID, existing.Title), nil
		})
}
