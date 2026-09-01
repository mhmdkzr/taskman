// Package app holds the shared runtime dependencies (App.Deps), configuration
// (App.Cfg), and HTTP mux (App.Mux) that slices register against.
package app

import (
	"net/http"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/mhmdkzr/taskman/internal/config"
)

// App bundles the application dependencies, configuration, and HTTP mux.
type App struct {
	Deps Deps
	Cfg  config.Config
	Mux  *http.ServeMux
}

// Deps holds the shared runtime dependencies of the application.
type Deps struct {
	NC *nats.Conn
	JS jetstream.JetStream
}
