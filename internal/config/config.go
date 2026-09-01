// Package config loads and validates service configuration.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"

	"github.com/mhmdkzr/taskman/pkg/logger"
)

// DefaultDBPath is where the agent's SQLite store lives when AGENT_DB_PATH is
// not set. The leading ~ is expanded to the user's home directory at load time.
const DefaultDBPath = "~/.taskman/taskman.db"

// Config is the top-level application configuration loaded from environment variables.
type Config struct {
	NATS   NATSConfig    `envPrefix:"NATS_"`
	Server ServerConfig  `envPrefix:"SERVER_"`
	Logger logger.Config `envPrefix:"LOGGER_"`
	Agent  AgentConfig   `envPrefix:"AGENT_"`
}

type NATSConfig struct {
	URL string `env:"URL"`
}

type ServerConfig struct {
	BindAddr        string        `env:"BIND_ADDR"`
	BasePath        string        `env:"BASE_PATH"`
	Timeout         time.Duration `env:"TIMEOUT"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT"`
}

// AgentConfig configures the agent runtime: provider credentials, the SQLite
// store path and the telegram tool credentials. Only the provider credentials
// are required; the rest fall back to defaults.
type AgentConfig struct {
	Provider        ProviderConfig `envPrefix:"PROVIDER_"`
	DBPath          string         `env:"DB_PATH" envDefault:"~/.taskman/taskman.db"`
	Model           string         `env:"MODEL" envDefault:"deepseek-v4-flash"`
	ReasoningEffort string         `env:"REASONING_EFFORT" envDefault:"medium"`
	MaxSteps        int            `env:"MAX_STEPS" envDefault:"100"`
	Telegram        Telegram       `envPrefix:"TELEGRAM_"`
}

// ProviderConfig holds the LLM provider credentials.
type ProviderConfig struct {
	BaseURL string `env:"BASE_URL"`
	APIKey  string `env:"API_KEY"`
}

// Telegram holds the credentials of the telegram_send / telegram_read tools.
type Telegram struct {
	APIKey    string `env:"API_KEY" envDefault:""`
	ChannelID string `env:"CHANNEL_ID" envDefault:""`
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
	if err := cfg.Agent.validate(); err != nil {
		return fmt.Errorf("agent: %w", err)
	}
	return nil
}

// AgentOptions returns the normalized agent configuration: defaults applied and
// any leading ~ in the DB path expanded to the user's home directory.
func (cfg *Config) AgentOptions() (AgentConfig, error) {
	a := cfg.Agent
	if a.DBPath == "" {
		a.DBPath = DefaultDBPath
	}
	dbPath, err := ExpandHome(a.DBPath)
	if err != nil {
		return AgentConfig{}, err
	}
	a.DBPath = dbPath
	return a, nil
}

// ExpandHome replaces a leading ~ with the user's home directory.
func ExpandHome(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("expand home dir: %w", err)
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
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

func (c AgentConfig) validate() error {
	if c.Provider.BaseURL == "" {
		return fmt.Errorf("PROVIDER_BASE_URL must not be empty")
	}
	if c.Provider.APIKey == "" {
		return fmt.Errorf("PROVIDER_API_KEY must not be empty")
	}
	return nil
}
