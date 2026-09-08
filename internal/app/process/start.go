package process

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	_ "modernc.org/sqlite" // Register the SQLite database driver.

	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/app/config"
	"github.com/mhmdkzr/loop/internal/app/register"
	"github.com/mhmdkzr/loop/internal/store"
	"github.com/mhmdkzr/loop/migrations"
	"github.com/mhmdkzr/loop/pkg/logger"
	"github.com/mhmdkzr/loop/pkg/middleware"
	"github.com/mhmdkzr/loop/pkg/middleware/logging"
	"github.com/mhmdkzr/loop/pkg/middleware/timeout"
	"github.com/mhmdkzr/loop/pkg/migrate"
)

// StartOptions configures application startup.
type StartOptions struct {
	AddrOverride   string
	PortOverride   int
	DBPathOverride string
	EnvFile        string
	LogLevel       string
	LogFormat      string
}

// Start boots the application, loads configuration, connects dependencies, and starts the HTTP server.
func Start(ctx context.Context, options StartOptions) error {
	slog.Info("starting application")

	var cfg config.Config
	if err := cfg.LoadFrom(options.EnvFile); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if options.LogLevel != "" {
		if err := cfg.Logger.Level.UnmarshalText([]byte(options.LogLevel)); err != nil {
			return fmt.Errorf("parse log level: %w", err)
		}
	}
	if options.LogFormat != "" {
		cfg.Logger.Format = logger.LogFormat(options.LogFormat)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	if err := logger.Init(cfg.Logger); err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}
	if options.DBPathOverride != "" {
		cfg.Database.Path = options.DBPathOverride
	}
	if options.AddrOverride != "" || options.PortOverride != 0 {
		bindHost, bindPort, err := net.SplitHostPort(cfg.Server.BindAddr)
		if err != nil {
			return fmt.Errorf("override server address: parse bind address: %w", err)
		}
		if options.AddrOverride != "" {
			bindHost = options.AddrOverride
		}
		if options.PortOverride != 0 {
			bindPort = strconv.Itoa(options.PortOverride)
		}
		cfg.Server.BindAddr = net.JoinHostPort(bindHost, bindPort)
	}

	db, err := store.Open(ctx, cfg.Database.Path)
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
		if err := runMigrations(ctx, db); err != nil {
			return fmt.Errorf("migrate db: %w", err)
		}
	}

	a := app.App{
		Deps: app.Deps{
			Store: db,
		},
		Cfg: cfg,
		Mux: http.NewServeMux(),
	}

	register.RegisterRoutes(a)
	applicationHandler := middleware.Chain(a.Mux,
		timeout.New(cfg.Server.Timeout),
		logging.New(),
	)
	httpServer, httpErrCh := startHTTPServer(applicationHandler, cfg.Server)

	slog.Info("application is ready")

	select {
	case <-ctx.Done():
	case err := <-httpErrCh:
		return err
	}

	slog.Info("shutdown")
	shutdownCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		cfg.Server.ShutdownTimeout,
	)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	return nil
}

// startHTTPServer starts the application HTTP server in the background,
// reporting any ListenAndServe failure (other than a graceful shutdown) on
// the returned channel.
func startHTTPServer(handler http.Handler, cfg config.ServerConfig) (*http.Server, <-chan error) {
	httpServer := &http.Server{
		Addr:              cfg.BindAddr,
		ReadHeaderTimeout: cfg.Timeout,
		ReadTimeout:       cfg.Timeout,
		WriteTimeout:      cfg.Timeout,
		IdleTimeout:       cfg.Timeout,
		MaxHeaderBytes:    1 << 20,
		Handler:           handler,
	}
	httpErrCh := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", cfg.BindAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpErrCh <- fmt.Errorf("listen and serve http: %w", err)
		}
	}()
	return httpServer, httpErrCh
}

// runMigrations runs all pending migrations against the application database.
func runMigrations(ctx context.Context, st *store.Store) error {
	slog.Info("running migrations")
	if err := migrate.Migrate(ctx, st.RW(), migrations.GetMigrationsFS()); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	if err := migrate.EnsureColumn(ctx, st.RW(), "tasks", "branch", "TEXT"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	if err := migrate.EnsureColumn(ctx, st.RW(), "tasks", "reasoning_effort", "TEXT"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	if err := migrate.EnsureColumn(ctx, st.RW(), "tasks", "failure_reason", "TEXT"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	if err := migrate.EnsureColumn(ctx, st.RW(), "tasks", "pipeline_step", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	slog.Info("migrations completed")
	return nil
}
