package register

import (
	"go.temporal.io/sdk/worker"

	"github.com/mhmdkzr/app/internal/app"
)

// RegisterEvents registers event-driven handlers on the Temporal worker.
func RegisterEvents(_ worker.Worker, _ app.App) {
}
