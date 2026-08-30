package process

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	tigerbeetle "github.com/tigerbeetle/tigerbeetle-go"
	mail "github.com/wneessen/go-mail"
	zitadelclient "github.com/zitadel/zitadel-go/v3/pkg/client"
	zitadel "github.com/zitadel/zitadel-go/v3/pkg/zitadel"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/auth/bff"
	"github.com/mhmdkzr/app/internal/auth/loginui"
	authregister "github.com/mhmdkzr/app/internal/auth/register"
	"github.com/mhmdkzr/app/internal/config"
	zitadelnotifications "github.com/mhmdkzr/app/internal/notifications/zitadel"
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

	if err := startNotifier(ctx, nc, cfg.Notifier); err != nil {
		return err
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

	sessions, sessionStore := newSessionManager(db, cfg.Auth)
	defer sessionStore.StopCleanup()

	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("jetstream init: %w", err)
	}

	auditlogCfg := auditlog.Config{Subject: "api.audit", Stream: "API_AUDIT"}
	if err := auditlog.CreateStream(ctx, js, auditlogCfg); err != nil {
		return fmt.Errorf("create audit stream: %w", err)
	}

	if err := streams.CreateStreams(ctx, js); err != nil {
		return fmt.Errorf("create streams: %w", err)
	}
	slog.Info("jetstream initialized and streams created")

	t, err := connectTemporal(ctx, cfg.Temporal)
	if err != nil {
		return err
	}
	defer t.Close()
	slog.Info("temporal connected and healthy", "host", cfg.Temporal.Host)

	tb, err := connectTigerBeetle(cfg.TigerBeetle)
	if err != nil {
		return err
	}
	defer tb.Close()

	s3Client, err := connectRustFS(ctx, cfg.RustFS)
	if err != nil {
		return err
	}
	slog.Info("rustfs connected", "endpoint", cfg.RustFS.Endpoint, "bucket", cfg.RustFS.Bucket)

	zc, err := connectZitadel(ctx, cfg.Zitadel)
	if err != nil {
		return err
	}
	defer func() {
		if err := zc.Close(); err != nil {
			slog.Error("close zitadel client", "error", err)
		}
	}()

	authService, err := newAuthService(ctx, cfg.Auth, db, sessions)
	if err != nil {
		return err
	}

	mailer, err := mail.NewClient(
		cfg.SMTP.Host,
		mail.WithPort(cfg.SMTP.Port),
		mail.WithTLSPolicy(mail.NoTLS),
	)
	if err != nil {
		return fmt.Errorf("create mailer: %w", err)
	}
	defer func() {
		if err := mailer.Close(); err != nil {
			slog.Error("close mailer", "error", err)
		}
	}()

	loginUIService, err := newLoginUIService(cfg.Auth, cfg.SMTP, sessions, authService, mailer)
	if err != nil {
		return err
	}

	a := app.App{
		Deps: app.Deps{
			DB:          db,
			NC:          nc,
			JS:          js,
			Temporal:    t,
			TigerBeetle: tb,
			RustFS:      s3Client,
			Zitadel:     zc,
			Mailer:      mailer,
			Sessions:    sessions,
		},
		Cfg: cfg,
		Mux: http.NewServeMux(),
	}

	w := worker.New(t, app.TemporalTaskQueue, worker.Options{})

	register.RegisterRoutes(a)
	authregister.RegisterRoutes(a, authService, loginUIService)
	register.RegisterActivities(w, a)
	register.RegisterWorkflows(w, a)
	register.RegisterEvents(w, a)
	slog.Info("temporal worker registered", "task_queue", app.TemporalTaskQueue)

	redactor, err := auditlog.NewWithRedactor(a.Deps.JS, auditlogCfg, auditlog.AuditEvent)
	if err != nil {
		return fmt.Errorf("auditlog init: %w", err)
	}
	applicationHandler := sessions.LoadAndSave(middleware.Chain(a.Mux,
		timeout.New(cfg.Server.Timeout),
		clientip.New(),
		redactor,
		logging.New(),
	))
	handler := withWebhooks(applicationHandler, mailer, cfg.Webhooks, cfg.SMTP)
	httpServer, httpErrCh := startHTTPServer(handler, cfg.Server)

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

// newSessionManager builds the PostgreSQL-backed session manager shared by
// the app and the BFF auth service's own transaction store.
func newSessionManager(
	db *sql.DB,
	cfg config.AuthConfig,
) (*scs.SessionManager, *postgresstore.PostgresStore) {
	sessionStore := postgresstore.New(db)
	sessions := scs.New()
	sessions.Store = sessionStore
	sessions.HashTokenInStore = true
	sessions.Cookie.Name = "app_session"
	sessions.Cookie.HttpOnly = true
	sessions.Cookie.SameSite = http.SameSiteStrictMode
	sessions.Cookie.Secure = cfg.CookieSecure
	sessions.Cookie.Path = "/"
	if cfg.Enabled {
		sessions.Lifetime = cfg.SessionLifetime
		sessions.IdleTimeout = cfg.SessionIdleTimeout
	}
	return sessions, sessionStore
}

// connectTemporal dials the Temporal frontend and verifies it is healthy.
func connectTemporal(
	ctx context.Context,
	cfg config.TemporalConfig,
) (temporalclient.Client, error) {
	t, err := temporalclient.Dial(temporalclient.Options{
		HostPort:  cfg.Host,
		Namespace: cfg.Namespace,
		Logger:    logger.NewTemporalLogger(slog.Default()),
	})
	if err != nil {
		return nil, fmt.Errorf("dial temporal client: %w", err)
	}
	if _, err := t.CheckHealth(ctx, &temporalclient.CheckHealthRequest{}); err != nil {
		t.Close()
		return nil, fmt.Errorf("check temporal health: %w", err)
	}
	return t, nil
}

// connectTigerBeetle connects to the TigerBeetle cluster and verifies it is healthy.
func connectTigerBeetle(cfg config.TigerBeetleConfig) (tigerbeetle.Client, error) {
	tb, err := tigerbeetle.NewClient(tigerbeetle.ToUint128(cfg.ClusterID), []string{cfg.Address})
	if err != nil {
		return nil, fmt.Errorf("connect tigerbeetle: %w", err)
	}
	if err := tb.Nop(); err != nil {
		tb.Close()
		return nil, fmt.Errorf("check tigerbeetle health: %w", err)
	}
	return tb, nil
}

// connectRustFS builds an S3 client for the RustFS object storage server.
// RustFS is S3-compatible (see https://docs.rustfs.com/en/developer/sdk/go)
// and is accessed via the AWS SDK for Go v2 with path-style addressing and a
// custom BaseEndpoint. It verifies connectivity with ListBuckets and ensures
// the configured bucket exists (creating it if missing), mirroring the
// TigerBeetle Nop / Temporal CheckHealth pattern.
func connectRustFS(ctx context.Context, cfg config.RustFSConfig) (*s3.Client, error) {
	awsCfg := aws.Config{
		Region: cfg.Region,
		Credentials: aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = cfg.UsePathStyle
	})

	// Verify connectivity and ensure bucket exists with a bounded timeout.
	hctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := client.ListBuckets(hctx, &s3.ListBucketsInput{}); err != nil {
		return nil, fmt.Errorf("check rustfs health (ListBuckets): %w", err)
	}

	if _, err := client.HeadBucket(hctx, &s3.HeadBucketInput{Bucket: aws.String(cfg.Bucket)}); err != nil {
		if _, cerr := client.CreateBucket(hctx, &s3.CreateBucketInput{Bucket: aws.String(cfg.Bucket)}); cerr != nil {
			// If creation fails because it already exists (race), treat as success.
			// Verify again with HeadBucket to confirm.
			if _, herr := client.HeadBucket(hctx, &s3.HeadBucketInput{Bucket: aws.String(cfg.Bucket)}); herr != nil {
				return nil, fmt.Errorf("ensure rustfs bucket %q: %w", cfg.Bucket, cerr)
			}
		} else {
			slog.Info("rustfs bucket created", "bucket", cfg.Bucket)
		}
	}

	return client, nil
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

// startNotifier validates and starts the optional Telegram notifier.
func startNotifier(ctx context.Context, nc *nats.Conn, cfg notifier.Config) error {
	if !cfg.Enabled {
		return nil
	}
	if cfg.Telegram.BotToken == "" {
		return fmt.Errorf("notifier is enabled but telegram bot token is not provided")
	}
	if cfg.Telegram.ChannelID == 0 {
		return fmt.Errorf("notifier is enabled but telegram channel id is not provided")
	}
	if err := notifier.Start(ctx, nc, cfg); err != nil {
		return fmt.Errorf("start notifier: %w", err)
	}
	slog.Info("notifier started")
	return nil
}

// connectZitadel dials the Zitadel API server used for management/admin calls.
func connectZitadel(ctx context.Context, cfg config.ZitadelConfig) (*zitadelclient.Client, error) {
	zitadelHost, zitadelPort, err := net.SplitHostPort(cfg.Domain)
	if err != nil {
		return nil, fmt.Errorf("parse zitadel domain: %w", err)
	}
	zitadelOptions := []zitadel.Option{zitadel.WithInsecure(zitadelPort)}
	if cfg.InstanceHost != "" {
		zitadelOptions = append(
			zitadelOptions,
			zitadel.WithTransportHeader("x-zitadel-instance-host", cfg.InstanceHost),
		)
	}
	zc, err := zitadelclient.New(ctx, zitadel.New(zitadelHost, zitadelOptions...))
	if err != nil {
		return nil, fmt.Errorf("connect zitadel: %w", err)
	}
	return zc, nil
}

// newAuthService initializes BFF authentication when enabled, returning nil otherwise.
func newAuthService(
	ctx context.Context,
	cfg config.AuthConfig,
	db *sql.DB,
	sessions *scs.SessionManager,
) (*bff.Service, error) {
	if !cfg.Enabled {
		return nil, nil //nolint:nilnil // disabled auth is a valid state, not an error
	}
	authService, err := bff.New(ctx, cfg, db, sessions)
	if err != nil {
		return nil, fmt.Errorf("initialize BFF authentication: %w", err)
	}
	return authService, nil
}

// newLoginUIService initializes the custom Session-API login UI when the BFF
// is enabled, returning nil otherwise. It reuses the BFF's own HTTP client
// (which honors the private Compose address override) and issuer.
//
// Session/OIDC-auth-request calls use cfg.LoginClientPATPath's token, which
// must belong to a ZITADEL account granted the IAM_LOGIN_CLIENT role; in
// local Compose it's the PAT ZITADEL's FirstInstance.Org.LoginClient
// bootstrap writes to the shared bootstrap volume for exactly this purpose.
// Registration/password-reset/TOTP-enrollment calls use the broader
// cfg.AdminPATPath token instead, since those exceed IAM_LOGIN_CLIENT's
// scope. See internal/auth/loginui/README.md.
func newLoginUIService(
	cfg config.AuthConfig,
	smtp config.SMTPConfig,
	sessions *scs.SessionManager,
	authService *bff.Service,
	mailer *mail.Client,
) (*loginui.Service, error) {
	if authService == nil {
		return nil, nil //nolint:nilnil // disabled auth is a valid state, not an error
	}
	loginToken, err := os.ReadFile(cfg.LoginClientPATPath)
	if err != nil {
		return nil, fmt.Errorf("read login client PAT: %w", err)
	}
	adminToken, err := os.ReadFile(cfg.AdminPATPath)
	if err != nil {
		return nil, fmt.Errorf("read admin PAT: %w", err)
	}
	return loginui.New(
		cfg.CookieSecure,
		authService.HTTPClient(),
		authService.Issuer(),
		loginui.StaticToken(strings.TrimSpace(string(loginToken))),
		loginui.StaticToken(strings.TrimSpace(string(adminToken))),
		loginui.NewSMTPMailer(mailer, smtp.From, smtp.FromName),
		strings.TrimSuffix(cfg.PostLogoutRedirectURL, "/"),
		sessions,
		authService,
	), nil
}

// withWebhooks routes the private /webhooks/ prefix to the Zitadel webhook
// handler when configured, falling back to applicationHandler otherwise.
func withWebhooks(
	applicationHandler http.Handler,
	mailer *mail.Client,
	webhooks config.WebhooksConfig,
	smtp config.SMTPConfig,
) http.Handler {
	if webhooks.ZitadelPathSecret == "" {
		return applicationHandler
	}
	webhookMux := http.NewServeMux()
	zitadelnotifications.NewHandler(webhooks.ZitadelPathSecret, mailer, smtp).Register(webhookMux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/webhooks/") {
			webhookMux.ServeHTTP(w, r)
			return
		}
		applicationHandler.ServeHTTP(w, r)
	})
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
