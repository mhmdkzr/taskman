// Package config loads and validates core service configuration.
package config

import (
	"fmt"
	"log/slog"
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
	NATS     NATSConfig      `envPrefix:"NATS_"`
	Server   ServerConfig    `envPrefix:"SERVER_"`
	Logger   logger.Config   `envPrefix:"LOGGER_"`
	Database pg.Config       `envPrefix:"POSTGRES_"`
	Temporal TemporalConfig  `envPrefix:"TEMPORAL_"`
	AuditLog AuditLogConfig  `envPrefix:"AUDIT_LOG_"`
	Notifier notifier.Config `envPrefix:"NOTIFIER_"`
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
