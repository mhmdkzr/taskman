// Package config loads and validates service configuration.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

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
	Telegram TelegramConfig `envPrefix:"TELEGRAM_"`
	Tavily   TavilyConfig   `envPrefix:"TAVILY_"`
	Browser  BrowserConfig  `envPrefix:"browser_"`
}

type ProviderConfig struct {
	BaseURL         string `env:"BASE_URL"`
	APIKeyOpenCode  string `env:"API_KEY_OPENCODE"`
	Model           string `env:"MODEL"`
	ReasoningEffort string `env:"REASONING_EFFORT"`
}

// TelegramConfig configures the telegram_send and telegram_read agent tools.
type TelegramConfig struct {
	APIKey    string `env:"API_KEY"`
	ChannelID string `env:"CHANNEL_ID"`
}

// TavilyConfig configures the websearch agent tool.
type TavilyConfig struct {
	APIKey string `env:"API_KEY"`
}

// BrowserConfig configures the browser automation agent tools backed by go-rod.
// The tools are enabled by default; set AGENT_BROWSER_ENABLED=false to disable.
type BrowserConfig struct {
	Enabled  bool          `env:"ENABLED"  envDefault:"true"`
	Headless bool          `env:"HEADLESS" envDefault:"true"`
	Bin      string        `env:"BIN"      envDefault:""`
	CDPURL   string        `env:"CDP_URL"  envDefault:""`
	Timeout  time.Duration `env:"TIMEOUT"  envDefault:"60s"`
}

type SQLiteConfig struct {
	Path        string `env:"PATH"         envDefault:":memory:"`
	AutoMigrate bool   `env:"AUTO_MIGRATE" envDefault:"true"`
}

func (c SQLiteConfig) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("PATH must not be empty")
	}
	return nil
}

type ServerConfig struct {
	BindAddr        string        `env:"BIND_ADDR"`
	BasePath        string        `env:"BASE_PATH"`
	Timeout         time.Duration `env:"TIMEOUT"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT"`
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
	if err := cfg.Telegram.validate(); err != nil {
		return fmt.Errorf("telegram: %w", err)
	}
	if err := cfg.Tavily.validate(); err != nil {
		return fmt.Errorf("tavily: %w", err)
	}
	if err := cfg.Browser.validate(); err != nil {
		return fmt.Errorf("browser: %w", err)
	}
	return nil
}

func (c ServerConfig) validate() error {
	if c.BindAddr == "" {
		return fmt.Errorf("BIND_ADDR must not be empty")
	}
	return nil
}

func (c ProviderConfig) validate() error {
	for name, value := range map[string]string{
		"BASE_URL": c.BaseURL, "API_KEY_OPENCODE": c.APIKeyOpenCode, "MODEL": c.Model, "REASONING_EFFORT": c.ReasoningEffort,
	} {
		if value == "" {
			return fmt.Errorf("%s must not be empty", name)
		}
	}
	return nil
}

// validate reports whether the telegram tool config is well-formed. The
// credentials are required: config loading fails when they are absent.
func (c TelegramConfig) validate() error {
	for name, value := range map[string]string{
		"API_KEY": c.APIKey, "CHANNEL_ID": c.ChannelID,
	} {
		if value == "" {
			return fmt.Errorf("%s must not be empty", name)
		}
	}
	return nil
}

// validate reports whether the websearch tool config is well-formed. The API
// key is required: config loading fails when it is absent.
func (c TavilyConfig) validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API_KEY must not be empty")
	}
	return nil
}

func (c BrowserConfig) validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf("TIMEOUT must be positive")
	}
	return nil
}
