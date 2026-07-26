package register

import (
	"go.temporal.io/sdk/worker"

	"github.com/mhmdkzr/app/internal/app"
)

// RegisterWorkflows registers all module-level Temporal workflows on the worker.
func RegisterWorkflows(_ worker.Worker, _ app.App) {
}
