// Package migrate owns the one-shot task schema migration command.
package migrate

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/migration/taskv1"
)

type Request struct {
	DryRun bool `json:"dry_run,omitempty" jsonschema:"validate and show migrations without rewriting task files"`
}

type Result = taskv1.Result

func Migrate(tasksDir string, req Request) (Result, error) {
	result, err := taskv1.Migrate(tasksDir, req.DryRun)
	if err != nil {
		return Result{}, fmt.Errorf("migrate tasks: %w", err)
	}
	return result, nil
}
