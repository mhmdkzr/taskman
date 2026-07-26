package register

import (
	"go.temporal.io/sdk/worker"

	"github.com/mhmdkzr/app/internal/app"
)

// RegisterActivities registers all module-level Temporal activities on the worker.
func RegisterActivities(_ worker.Worker, _ app.App) {
}
