// Package config loads and validates service configuration.
package config

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"

	"github.com/mhmdkzr/app/pkg/logger"
	"github.com/mhmdkzr/app/pkg/notifier"
	"github.com/mhmdkzr/app/pkg/pg"
)

// Config is the top-level application configuration loaded from environment variables.
type Config struct {
	NATS        NATSConfig        `envPrefix:"NATS_"`
	Server      ServerConfig      `envPrefix:"SERVER_"`
	Logger      logger.Config     `envPrefix:"LOGGER_"`
	Database    pg.Config         `envPrefix:"POSTGRES_"`
	Temporal    TemporalConfig    `envPrefix:"TEMPORAL_"`
	AuditLog    AuditLogConfig    `envPrefix:"AUDIT_LOG_"`
	Notifier    notifier.Config   `envPrefix:"NOTIFIER_"`
	TigerBeetle TigerBeetleConfig `envPrefix:"TIGERBEETLE_"`
	Zitadel     ZitadelConfig     `envPrefix:"ZITADEL_CLIENT_"`
	Auth        AuthConfig        `envPrefix:"AUTH_"`
	Webhooks    WebhooksConfig    `envPrefix:"WEBHOOKS_"`
	SMTP        SMTPConfig        `envPrefix:"SMTP_"`
}

type NATSConfig struct {
	URL string `env:"URL"`
}

type AuditLogConfig struct {
	Timeout time.Duration `env:"TIMEOUT"`
}

type ServerConfig struct {
	BindAddr        string        `env:"BIND_ADDR"`
	BasePath        string        `env:"BASE_PATH"`
	Timeout         time.Duration `env:"TIMEOUT"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT"`
}

type TemporalConfig struct {
	Host      string `env:"HOST"`
	Namespace string `env:"NAMESPACE"`
}

type TigerBeetleConfig struct {
	Address   string `env:"ADDRESS"    envDefault:"127.0.0.1:3000"`
	ClusterID uint64 `env:"CLUSTER_ID" envDefault:"0"`
}

type ZitadelConfig struct {
	Domain       string `env:"DOMAIN"        envDefault:"127.0.0.1:8080"`
	InstanceHost string `env:"INSTANCE_HOST" envDefault:""`
	Insecure     bool   `env:"INSECURE"      envDefault:"true"`
}

// AuthConfig configures the server-side OIDC client and its persistent browser sessions.
type AuthConfig struct {
	Enabled               bool          `env:"ENABLED"                  envDefault:"false"`
	Issuer                string        `env:"ISSUER"                   envDefault:""`
	InternalAddress       string        `env:"INTERNAL_ADDRESS"         envDefault:""`
	ClientID              string        `env:"CLIENT_ID"                envDefault:""`
	ClientSecret          string        `env:"CLIENT_SECRET"            envDefault:""`
	RedirectURL           string        `env:"REDIRECT_URL"             envDefault:""`
	PostLogoutRedirectURL string        `env:"POST_LOGOUT_REDIRECT_URL" envDefault:""`
	LoginClientPATPath    string        `env:"LOGIN_CLIENT_PAT_PATH"    envDefault:""`
	AdminPATPath          string        `env:"ADMIN_PAT_PATH"           envDefault:""`
	SessionLifetime       time.Duration `env:"SESSION_LIFETIME"         envDefault:"24h"`
	SessionIdleTimeout    time.Duration `env:"SESSION_IDLE_TIMEOUT"     envDefault:"8h"`
	RefreshLeeway         time.Duration `env:"REFRESH_LEEWAY"           envDefault:"1m"`
	CookieSecure          bool          `env:"COOKIE_SECURE"            envDefault:"true"`
}

// WebhooksConfig configures the listener that is reachable only from the private network.
type WebhooksConfig struct {
	ZitadelPathSecret string `env:"ZITADEL_PATH_SECRET" envDefault:""`
}

type SMTPConfig struct {
	Host     string `env:"HOST"      envDefault:"127.0.0.1"`
	Port     int    `env:"PORT"      envDefault:"1025"`
	From     string `env:"FROM"      envDefault:"noreply@example.com"`
	FromName string `env:"FROM_NAME" envDefault:"App"`
	Username string `env:"USERNAME"`
	Password string `env:"PASSWORD"`
}

// Load reads environment variables into cfg, first loading .env if SKIP_ENV_AUTO_LOAD is not set.
func (cfg *Config) Load() error {
	if strings.ToLower(os.Getenv("SKIP_ENV_AUTO_LOAD")) != "true" {
		if err := godotenv.Load(); err != nil {
			slog.Error("error loading .env file", "error", err.Error())
		}
	}

	opts := env.Options{RequiredIfNoDef: true}
	if err := env.ParseWithOptions(cfg, opts); err != nil {
		return fmt.Errorf("parse environment config: %w", err)
	}

	slog.Info("config loaded from environment variables")
	return nil
}

// Validate returns an error if any configuration section is invalid.
func (cfg *Config) Validate() error {
	if err := cfg.NATS.validate(); err != nil {
		return fmt.Errorf("nats: %w", err)
	}
	if err := cfg.Server.validate(); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	if err := cfg.Logger.Validate(); err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	if err := cfg.Database.Validate(); err != nil {
		return fmt.Errorf("database: %w", err)
	}
	if err := cfg.Temporal.validate(); err != nil {
		return fmt.Errorf("temporal: %w", err)
	}
	if err := cfg.AuditLog.validate(); err != nil {
		return fmt.Errorf("audit_log: %w", err)
	}
	if err := cfg.Notifier.Validate(); err != nil {
		return fmt.Errorf("notifier: %w", err)
	}
	if err := cfg.TigerBeetle.validate(); err != nil {
		return fmt.Errorf("tigerbeetle: %w", err)
	}
	if err := cfg.Zitadel.validate(); err != nil {
		return fmt.Errorf("zitadel: %w", err)
	}
	if err := cfg.Auth.validate(); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if err := cfg.Webhooks.validate(); err != nil {
		return fmt.Errorf("webhooks: %w", err)
	}
	if err := cfg.SMTP.validate(); err != nil {
		return fmt.Errorf("smtp: %w", err)
	}
	return nil
}

func (c AuthConfig) validate() error {
	if !c.Enabled {
		return nil
	}
	for name, value := range map[string]string{
		"ISSUER": c.Issuer, "CLIENT_ID": c.ClientID, "CLIENT_SECRET": c.ClientSecret,
		"REDIRECT_URL": c.RedirectURL, "POST_LOGOUT_REDIRECT_URL": c.PostLogoutRedirectURL,
		"LOGIN_CLIENT_PAT_PATH": c.LoginClientPATPath, "ADMIN_PAT_PATH": c.AdminPATPath,
	} {
		if value == "" {
			return fmt.Errorf("%s must not be empty when ENABLED", name)
		}
	}
	if c.SessionLifetime <= 0 || c.SessionIdleTimeout <= 0 || c.RefreshLeeway < 0 {
		return fmt.Errorf("session durations must be positive (refresh leeway may be zero)")
	}
	if c.InternalAddress != "" {
		if _, _, err := net.SplitHostPort(c.InternalAddress); err != nil {
			return fmt.Errorf("INTERNAL_ADDRESS must be host:port when ENABLED: %w", err)
		}
	}
	return nil
}

func (c WebhooksConfig) validate() error {
	return nil
}

func (c NATSConfig) validate() error {
	if c.URL == "" {
		return fmt.Errorf("URL must not be empty")
	}
	return nil
}

func (c ServerConfig) validate() error {
	if c.BindAddr == "" {
		return fmt.Errorf("BIND_ADDR must not be empty")
	}
	return nil
}

func (c TemporalConfig) validate() error {
	if c.Host == "" {
		return fmt.Errorf("HOST must not be empty")
	}
	return nil
}

func (c AuditLogConfig) validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf("TIMEOUT must be positive")
	}
	return nil
}

func (c TigerBeetleConfig) validate() error {
	if c.Address == "" {
		return fmt.Errorf("ADDRESS must not be empty")
	}
	return nil
}

func (c ZitadelConfig) validate() error {
	if c.Domain == "" {
		return fmt.Errorf("DOMAIN must not be empty")
	}
	return nil
}

func (c SMTPConfig) validate() error {
	if c.Host == "" {
		return fmt.Errorf("HOST must not be empty")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("PORT %d is outside range [1, 65535]", c.Port)
	}
	if c.From == "" {
		return fmt.Errorf("FROM must not be empty")
	}
	if !strings.Contains(c.From, "@") {
		return fmt.Errorf("FROM %q must contain @", c.From)
	}
	return nil
}
