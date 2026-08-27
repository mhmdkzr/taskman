package process

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/config"
	"github.com/mhmdkzr/app/internal/register"
	"github.com/mhmdkzr/app/internal/streams"
	"github.com/mhmdkzr/app/migrations"
	"github.com/mhmdkzr/app/pkg/logger"
	"github.com/mhmdkzr/app/pkg/middleware"
	"github.com/mhmdkzr/app/pkg/middleware/auditlog"
	"github.com/mhmdkzr/app/pkg/middleware/clientip"
	"github.com/mhmdkzr/app/pkg/middleware/logging"
	"github.com/mhmdkzr/app/pkg/middleware/timeout"
	"github.com/mhmdkzr/app/pkg/migrate"
	"github.com/mhmdkzr/app/pkg/notifier"
	"github.com/mhmdkzr/app/pkg/pg"
)

// Start boots the application: loads config, connects dependencies, starts the HTTP server and Temporal worker.
func Start(ctx context.Context) error {
	slog.Info("starting application")

	var cfg config.Config
	if err := cfg.Load(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	nc, err := nats.Connect(cfg.NATS.URL)
	if err != nil {
		return fmt.Errorf("connect nats: %w", err)
	}
	defer nc.Close()
	slog.Info("nats connected", "url", cfg.NATS.URL)

	if err := logger.Init(cfg.Logger, nc); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	slog.Info("logger initialized")

	if cfg.Notifier.Enabled {
		if cfg.Notifier.Telegram.BotToken == "" {
			return fmt.Errorf("notifier is enabled but telegram bot token is not provided")
		}
		if cfg.Notifier.Telegram.ChannelID == 0 {
			return fmt.Errorf("notifier is enabled but telegram channel id is not provided")
		}
		if err := notifier.Start(ctx, nc, cfg.Notifier); err != nil {
			return fmt.Errorf("start notifier: %w", err)
		}
		slog.Info("notifier started")
	}

	db, err := pg.Open(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("close database", "error", err)
		}
	}()
	slog.Info("database connected")

	slog.Info("auto migrate", "enabled", cfg.Database.AutoMigrate)
	if cfg.Database.AutoMigrate {
		if err := runMigrations(ctx, cfg.Database); err != nil {
			return fmt.Errorf("migrate db: %w", err)
		}
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("jetstream init: %w", err)
	}

	if err := streams.CreateStreams(ctx, js); err != nil {
		return fmt.Errorf("create streams: %w", err)
	}
	slog.Info("jetstream initialized and streams created")

	t, err := temporalclient.Dial(temporalclient.Options{
		HostPort:  cfg.Temporal.Host,
		Namespace: cfg.Temporal.Namespace,
		Logger:    logger.NewTemporalLogger(slog.Default()),
	})
	if err != nil {
		return fmt.Errorf("dial temporal client: %w", err)
	}
	defer t.Close()

	if _, err := t.CheckHealth(ctx, &temporalclient.CheckHealthRequest{}); err != nil {
		return fmt.Errorf("check temporal health: %w", err)
	}
	slog.Info("temporal connected and healthy", "host", cfg.Temporal.Host)

	a := app.App{
		Deps: app.Deps{
			DB:       db,
			NC:       nc,
			JS:       js,
			Temporal: t,
		},
		Cfg: cfg,
		Mux: http.NewServeMux(),
	}

	w := worker.New(t, app.TemporalTaskQueue, worker.Options{})

	register.RegisterRoutes(a)
	register.RegisterActivities(w, a)
	register.RegisterWorkflows(w, a)
	register.RegisterEvents(w, a)
	slog.Info("temporal worker registered", "task_queue", app.TemporalTaskQueue)

	redactor, err := auditlog.NewWithRedactor(a.Deps.JS, auditlog.Config{}, auditlog.AuditEvent)
	if err != nil {
		return fmt.Errorf("auditlog init: %w", err)
	}

	httpServer := &http.Server{
		Addr:              cfg.Server.BindAddr,
		ReadHeaderTimeout: cfg.Server.Timeout,
		Handler: middleware.Chain(a.Mux,
			timeout.New(cfg.Server.Timeout),
			clientip.New(),
			redactor,
			logging.New(),
		),
	}
	httpErrCh := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", cfg.Server.BindAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpErrCh <- fmt.Errorf("listen and serve http: %w", err)
		}
	}()

	if err := w.Start(); err != nil {
		return fmt.Errorf("start temporal worker: %w", err)
	}
	defer w.Stop()
	slog.Info("temporal worker started")

	if err := startRuntimeProcesses(ctx, a, cfg); err != nil {
		return err
	}

	slog.Info("application is ready")

	select {
	case <-ctx.Done():
	case err := <-httpErrCh:
		return err
	}

	slog.Info("shutdown")
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	return nil
}

func startRuntimeProcesses(ctx context.Context, a app.App, cfg config.Config) error {
	c := auditlog.Config{Subject: "api.audit", Stream: "API_AUDIT"}
	if err := auditlog.Start(ctx, a.Deps.JS, a.Deps.DB, c, cfg.AuditLog.Timeout); err != nil {
		return fmt.Errorf("start audit log consumer: %w", err)
	}
	slog.Info("audit log consumer started")

	return nil
}

// runMigrations opens a database connection and runs all pending migrations.
func runMigrations(ctx context.Context, cfg pg.Config) (err error) {
	slog.Info("running migrations")
	mdb, err := pg.Open(ctx, cfg)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer func() {
		if closeErr := mdb.Close(); closeErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; close migration db: %w", err, closeErr)
			} else {
				err = fmt.Errorf("close migration db: %w", closeErr)
			}
		}
	}()

	if err := migrate.Migrate(ctx, mdb, migrations.GetMigrationsFS()); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	slog.Info("migrations completed")
	return nil
}
