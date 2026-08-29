package app

import (
	"database/sql"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	mail "github.com/wneessen/go-mail"
	temporalclient "go.temporal.io/sdk/client"

	tigerbeetle "github.com/tigerbeetle/tigerbeetle-go"
	zitadelclient "github.com/zitadel/zitadel-go/v3/pkg/client"

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
	Zitadel     *zitadelclient.Client
	Mailer      *mail.Client
	Sessions    *scs.SessionManager
}

// TemporalTaskQueue is the default task queue name for all Temporal workflows.
const TemporalTaskQueue = "app"
