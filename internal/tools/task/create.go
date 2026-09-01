package task

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/taskman/internal/task"
)

type createInput struct {
	Title         string   `json:"title"                  jsonschema:"description=Short title (a few words), not a sentence."`
	TaskType      string   `json:"task_type"              jsonschema:"description=One of: bug, docs, feature, refactor, test."`
	Urgency       string   `json:"urgency"                jsonschema:"description=Time pressure: low, medium, or high."`
	Importance    string   `json:"importance"             jsonschema:"description=Impact if never done: low, medium, or high."`
	Risk          string   `json:"risk"                   jsonschema:"description=Blast radius if execution goes wrong: low, medium, or high (independent of urgency/importance)."`
	What          string   `json:"what"                   jsonschema:"description=What the task is."`
	Why           string   `json:"why"                    jsonschema:"description=Why it matters."`
	How           string   `json:"how"                    jsonschema:"description=How to do it."`
	Where         []string `json:"where,omitempty"        jsonschema:"description=file:line citations."`
	Invariants    []string `json:"invariants,omitempty"   jsonschema:"description=Things that must stay true after the change."`
	Packages      []string `json:"packages"               jsonschema:"description=Packages this task touches."`
	CompletedWhen []string `json:"completed_when"         jsonschema:"description=Independently checkable completion conditions."`
	Tags          []string `json:"tags,omitempty"`
	Dependencies  []string `json:"dependencies,omitempty" jsonschema:"description=IDs of other tasks this one depends on."`
	Variant       string   `json:"variant,omitempty"      jsonschema:"description=Model reasoning effort to run this task with: low, medium, high, xhigh, max, or default. Defaults to medium."`
}

// CreateTool returns the task_create tool bound to db.
func CreateTool(db *sql.DB) goai.Tool {
	return goai.NewTool("task_create",
		"File a new task in the backlog. Understand what's actually being asked before calling this — "+
			"don't transcribe a request literally; pick fields you've reasoned about, not fields you copied.",
		func(ctx context.Context, in createInput) (string, error) {
			variant := in.Variant
			if variant == "" {
				variant = "medium"
			}
			if err := validateSpec(in.TaskType, in.Urgency, in.Importance, in.Risk,
				in.Title, in.What, in.Why, in.How, in.Packages, in.CompletedWhen); err != nil {
				return "", err
			}
			if err := validateVariant(variant); err != nil {
				return "", err
			}
			t := task.Task{
				ID:            task.NewTaskID(),
				Title:         in.Title,
				TaskType:      in.TaskType,
				Urgency:       in.Urgency,
				Importance:    in.Importance,
				Risk:          in.Risk,
				What:          in.What,
				Why:           in.Why,
				How:           in.How,
				Where:         in.Where,
				Invariants:    in.Invariants,
				Packages:      in.Packages,
				CompletedWhen: in.CompletedWhen,
				Tags:          in.Tags,
				Dependencies:  in.Dependencies,
				Variant:       variant,
			}
			if err := task.CreateTask(ctx, db, t); err != nil {
				return "", fmt.Errorf("task_create: %w", err)
			}
			return fmt.Sprintf("created task %s: %s", t.ID, t.Title), nil
		})
}
