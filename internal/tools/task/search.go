package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/taskman/internal/task"
)

var validStatuses = []string{"created", "started", "completed", "reviewed"}

type searchInput struct {
	Status   string `json:"status,omitempty"    jsonschema:"description=Filter by status: created, started, completed, or reviewed. Omit for any status."`
	TaskType string `json:"task_type,omitempty" jsonschema:"description=Filter by task type: bug, docs, feature, refactor, or test."`
	Package  string `json:"package,omitempty"   jsonschema:"description=Filter to tasks whose packages list contains exactly this value."`
	Tag      string `json:"tag,omitempty"       jsonschema:"description=Filter to tasks whose tags list contains exactly this value."`
	Query    string `json:"query,omitempty"     jsonschema:"description=Substring match against title/what/why/how."`
}

// SearchTool returns the task_search tool bound to db.
func SearchTool(db *sql.DB) goai.Tool {
	return goai.NewTool("task_search",
		"Search the task backlog. Every set field narrows the result; leave fields empty to widen. "+
			"Returns a compact summary per task, not the full record — use task_get for that.",
		func(ctx context.Context, in searchInput) (string, error) {
			if in.Status != "" && !slices.Contains(validStatuses, in.Status) {
				return "", fmt.Errorf("invalid status %q: want one of %v", in.Status, validStatuses)
			}
			if in.TaskType != "" && !slices.Contains(validTaskTypes, in.TaskType) {
				return "", fmt.Errorf("invalid task_type %q: want one of %v", in.TaskType, validTaskTypes)
			}
			tasks, err := task.SearchTasks(ctx, db, task.TaskFilter{
				Status:   task.TaskStatus(in.Status),
				TaskType: in.TaskType,
				Package:  in.Package,
				Tag:      in.Tag,
				Query:    in.Query,
			})
			if err != nil {
				return "", fmt.Errorf("task_search: %w", err)
			}

			type summary struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Status   string `json:"status"`
				TaskType string `json:"task_type"`
				Urgency  string `json:"urgency"`
				Risk     string `json:"risk"`
			}
			out := make([]summary, len(tasks))
			for i, t := range tasks {
				out[i] = summary{
					ID:       t.ID.String(),
					Title:    t.Title,
					Status:   string(t.Status),
					TaskType: t.TaskType,
					Urgency:  t.Urgency,
					Risk:     t.Risk,
				}
			}
			b, err := json.Marshal(out)
			if err != nil {
				return "", fmt.Errorf("marshal search results: %w", err)
			}
			return string(b), nil
		})
}
