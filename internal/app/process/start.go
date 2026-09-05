package process

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	_ "modernc.org/sqlite" // Register the SQLite database driver.

	"github.com/mhmdkzr/loop/internal/agent"
	agenttools "github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
	"github.com/mhmdkzr/loop/internal/agent/tools/telegram"
	"github.com/mhmdkzr/loop/internal/agent/tools/websearch"
	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/app/config"
	"github.com/mhmdkzr/loop/internal/app/register"
	"github.com/mhmdkzr/loop/internal/rpc"
	"github.com/mhmdkzr/loop/migrations"
	"github.com/mhmdkzr/loop/pkg/logger"
	"github.com/mhmdkzr/loop/pkg/middleware"
	"github.com/mhmdkzr/loop/pkg/middleware/logging"
	"github.com/mhmdkzr/loop/pkg/middleware/timeout"
	"github.com/mhmdkzr/loop/pkg/migrate"
)

// Start boots the application: loads config, connects dependencies, starts the HTTP server.
type StartOptions struct {
	AddrOverride   string
	PortOverride   int
	DBPathOverride string
	EnvFile        string
	LogLevel       string
	LogFormat      string
	RPC            bool
}

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

	db, err := openDatabase(ctx, cfg.Database)
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
	if err := agent.Seed(ctx, db, cfg.Provider); err != nil {
		return fmt.Errorf("seed agent data: %w", err)
	}
	slog.Info("agent data seeded")

	a := app.App{
		Deps: app.Deps{
			DB: db,
		},
		Cfg: cfg,
		Mux: http.NewServeMux(),
	}
	telegramClient, err := telegram.NewClientFromConfig(cfg.Telegram)
	if err != nil {
		return fmt.Errorf("initialize telegram client: %w", err)
	}
	a.Deps.AgentTools = agenttools.Deps{
		DB: db, Config: cfg,
		Configured: agent.ConfiguredTools(browser.NewClientFromConfig(cfg.Browser), telegramClient, websearch.NewClientFromConfig(cfg.Tavily)),
	}

	register.RegisterRoutes(a)
	if options.RPC {
		a.Mux.Handle("POST /rpc", rpc.NewHandler(a))
	}
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
func runMigrations(ctx context.Context, db *sql.DB) error {
	slog.Info("running migrations")
	if err := migrate.Migrate(ctx, db, migrations.GetMigrationsFS()); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	slog.Info("migrations completed")
	return nil
}

func openDatabase(ctx context.Context, cfg config.SQLiteConfig) (*sql.DB, error) {
	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	if cfg.Path == ":memory:" {
		db.SetMaxOpenConns(1)
	}
	return db, nil
}
