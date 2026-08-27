package app

import (
	"database/sql"
	"net/http"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/mhmdkzr/app/internal/config"
)

// App bundles the application dependencies, configuration, and HTTP mux.
type App struct {
	Deps Deps
	Cfg  config.Config
	Mux  *http.ServeMux
}

// Deps holds the shared runtime dependencies of the application.
type Deps struct {
	DB       *sql.DB
	NC       *nats.Conn
	JS       jetstream.JetStream
	Temporal temporalclient.Client
}

// TemporalTaskQueue is the default task queue name for all Temporal workflows.
const TemporalTaskQueue = "app"
