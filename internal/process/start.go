package process

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/mhmdkzr/taskman/internal/agent"
	"github.com/mhmdkzr/taskman/internal/app"
	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/register"
	"github.com/mhmdkzr/taskman/internal/server"
	"github.com/mhmdkzr/taskman/internal/streams"
	"github.com/mhmdkzr/taskman/pkg/logger"
	"github.com/mhmdkzr/taskman/pkg/middleware"
	"github.com/mhmdkzr/taskman/pkg/middleware/clientip"
	"github.com/mhmdkzr/taskman/pkg/middleware/logging"
	"github.com/mhmdkzr/taskman/pkg/middleware/timeout"
)

// Start boots the application: loads config, connects NATS/JetStream, starts
// the HTTP server and the agent runtime (store, scheduler, consumer, request
// subscription) on the same connection.
func Start(ctx context.Context) error {
	slog.Info("starting application")

	var cfg config.Config
	if err := cfg.Load(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
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

	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("jetstream init: %w", err)
	}

	if err := streams.CreateStreams(ctx, js); err != nil {
		return fmt.Errorf("create streams: %w", err)
	}
	slog.Info("jetstream initialized and streams created")

	a := app.App{
		Deps: app.Deps{
			NC: nc,
			JS: js,
		},
		Cfg: cfg,
		Mux: http.NewServeMux(),
	}

	register.RegisterRoutes(a)

	applicationHandler := middleware.Chain(a.Mux,
		timeout.New(cfg.Server.Timeout),
		clientip.New(),
		logging.New(),
	)
	httpServer, httpErrCh := startHTTPServer(applicationHandler, cfg.Server)

	agentCfg, err := cfg.AgentOptions()
	if err != nil {
		return fmt.Errorf("agent config: %w", err)
	}
	agentErrCh := make(chan error, 1)
	if err := startAgent(ctx, nc, agentCfg, agentErrCh); err != nil {
		return err
	}

	slog.Info("application is ready")

	select {
	case <-ctx.Done():
	case err := <-httpErrCh:
		return err
	case err := <-agentErrCh:
		if err != nil {
			return err
		}
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

	// The agent runtime drains in-flight runs once its context is cancelled;
	// wait for it before releasing the store.
	if err := <-agentErrCh; err != nil {
		slog.Error("agent runtime", "error", err)
	}
	return nil
}

// startAgent assembles and runs the agent runtime: the SQLite store, the
// durable scheduler, the scheduled-run consumer and the request subscription.
// It reports fatal runtime errors on errCh; on ctx cancellation the runtime
// drains and reports nil.
func startAgent(ctx context.Context, nc *nats.Conn, cfg config.AgentConfig, errCh chan<- error) error {
	opts, err := agent.OptionsFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("agent options: %w", err)
	}
	pub := publisher.NewPublisher(nc)
	agentServer, err := server.New(opts, pub)
	if err != nil {
		return fmt.Errorf("agent server: %w", err)
	}
	go func() {
		errCh <- agentServer.Run(ctx, agentLogger())
		_ = agentServer.Close()
	}()
	return nil
}

// agentLogger returns a *log.Logger that routes the agent runtime's diagnostics
// through slog.
func agentLogger() *log.Logger {
	return slog.NewLogLogger(slog.Default().Handler(), slog.LevelInfo)
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
