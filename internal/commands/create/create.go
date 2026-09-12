// Package create owns the "create" command: it starts a new task.
package create

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is create's input.
type Request struct {
	Title       string            `json:"title,omitempty"  jsonschema:"short human-readable title"`
	Description string            `json:"description"      jsonschema:"what the task should accomplish"`
	Labels      map[string]string `json:"labels,omitempty" jsonschema:"labels as key/value pairs"`
}

func (r Request) validate() error {
	if r.Description == "" {
		return fmt.Errorf("description is required")
	}
	return nil
}

// Create starts a new task with a freshly generated id and returns it.
func Create(ctx context.Context, st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("create task: %w", err)
	}
	definition := task.TaskDefinition{Title: req.Title, Description: req.Description, Labels: req.Labels}
	t, err := st.Create(ctx, uuid.NewV7(), definition, time.Now().UTC())
	if err != nil {
		return task.Task{}, fmt.Errorf("create task: %w", err)
	}
	return t, nil
}
