// Package config loads and validates service configuration.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"

	"github.com/mhmdkzr/loop/pkg/logger"
)

// Config is the top-level application configuration loaded from environment variables.
type Config struct {
	Server   ServerConfig   `envPrefix:"SERVER_"`
	Logger   logger.Config  `envPrefix:"LOGGER_"`
	Database SQLiteConfig   `envPrefix:"SQLITE_"`
	Provider ProviderConfig `envPrefix:"PROVIDER_"`
}

// Validate returns an error if any configuration section is invalid.
func (cfg *Config) Validate() error {
	if err := cfg.Server.validate(); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	if err := cfg.Logger.Validate(); err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	if err := cfg.Database.Validate(); err != nil {
		return fmt.Errorf("database: %w", err)
	}
	if err := cfg.Provider.validate(); err != nil {
		return fmt.Errorf("provider: %w", err)
	}
	return nil
}

// Load reads environment variables into cfg, first loading .env if SKIP_ENV_AUTO_LOAD is not set.
func (cfg *Config) Load() error {
	return cfg.load("")
}

// LoadFrom reads environment variables into cfg, loading envFile first.
func (cfg *Config) LoadFrom(envFile string) error {
	return cfg.load(envFile)
}

func (cfg *Config) load(envFile string) error {
	if strings.ToLower(os.Getenv("SKIP_ENV_AUTO_LOAD")) != "true" {
		var err error
		if envFile == "" {
			err = godotenv.Load()
		} else {
			err = godotenv.Load(envFile)
		}
		if err != nil {
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
