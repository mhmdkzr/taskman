// Package app holds the shared runtime dependencies (App.Deps), configuration
// (App.Cfg), and HTTP mux (App.Mux) that slices register against.
package app

import (
	"database/sql"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	tigerbeetle "github.com/tigerbeetle/tigerbeetle-go"
	mail "github.com/wneessen/go-mail"
	zitadelclient "github.com/zitadel/zitadel-go/v3/pkg/client"
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
	DB          *sql.DB
	NC          *nats.Conn
	JS          jetstream.JetStream
	Temporal    temporalclient.Client
	TigerBeetle tigerbeetle.Client
	RustFS      *s3.Client
	Zitadel     *zitadelclient.Client
	Mailer      *mail.Client
	Sessions    *scs.SessionManager
}

// TemporalTaskQueue is the default task queue name for all Temporal workflows.
const TemporalTaskQueue = "app"
