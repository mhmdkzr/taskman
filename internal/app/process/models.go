package process

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/mhmdkzr/loop/internal/agent/models"
	"github.com/mhmdkzr/loop/internal/app/config"
	"github.com/mhmdkzr/loop/internal/store"
	"github.com/mhmdkzr/loop/pkg/logger"
)

// ModelsOptions configures the model catalog command.
type ModelsOptions struct {
	Refresh        bool
	List           bool
	Output         io.Writer
	DBPathOverride string
	EnvFile        string
	LogLevel       string
	LogFormat      string
	Provider       string
	JSON           bool
}

// Models runs the model catalog command. Exactly one of Refresh or List must
// be true. Refresh synchronizes the catalog; List writes the stored catalog.
func Models(ctx context.Context, options ModelsOptions) error {
	if options.Refresh == options.List {
		return fmt.Errorf("exactly one of refresh or list must be selected")
	}
	if options.Provider != "" && !options.Refresh {
		return fmt.Errorf("provider can only be used with refresh")
	}
	if options.Provider != "" && options.Provider != "opencode" {
		return fmt.Errorf("unsupported refresh provider %q", options.Provider)
	}

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

	db, err := store.Open(ctx, cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			// The command's primary operation has already returned; its close
			// failure cannot be propagated without masking that result.
			slog.Error("close models database", "error", closeErr)
		}
	}()

	if cfg.Database.AutoMigrate {
		if err := runMigrations(ctx, db); err != nil {
			return fmt.Errorf("migrate db: %w", err)
		}
	}
	if options.Refresh {
		return wrapModelCommandError("refresh models", models.Refresh(ctx, db, cfg.Provider))
	}
	return wrapModelCommandError("list models", models.List(ctx, db, options.Output, options.JSON))
}

func wrapModelCommandError(operation string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return nil
}
