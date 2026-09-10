// Package store persists task.Task values as YAML documents.
package store

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
)

type document struct {
	Task task.Task `yaml:"task"`
}

func newDocument(t task.Task) document {
	return document{Task: t}
}

func (d document) taskValue() (task.Task, error) {
	if err := task.Validate(d.Task); err != nil {
		return task.Task{}, fmt.Errorf("validate task: %w", err)
	}
	return d.Task, nil
}
