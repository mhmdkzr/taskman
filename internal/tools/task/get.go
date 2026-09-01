package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"uuid"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/taskman/internal/task"
)

type getInput struct {
	ID string `json:"id" jsonschema:"description=Task ID (UUID) to read in full."`
}

// GetTool returns the task_get tool bound to db.
func GetTool(db *sql.DB) goai.Tool {
	return goai.NewTool("task_get",
		"Read one task in full by id, including its status, session ids, token usage, and timestamps.",
		func(ctx context.Context, in getInput) (string, error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return "", fmt.Errorf("invalid id %q: %w", in.ID, err)
			}
			t, err := task.GetTask(ctx, db, id)
			if err != nil {
				return "", fmt.Errorf("task_get: %w", err)
			}
			b, err := json.Marshal(t)
			if err != nil {
				return "", fmt.Errorf("marshal task %s: %w", id, err)
			}
			return string(b), nil
		})
}
