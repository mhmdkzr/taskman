package logger

import (
	"fmt"
	"log/slog"
	"os"
)

// Init initializes the logger.
func Init(cfg Config) error {
	opts := &slog.HandlerOptions{Level: cfg.Level, AddSource: true}

	var baseHandler slog.Handler
	switch cfg.Format {
	case LogFormatText:
		baseHandler = slog.NewTextHandler(os.Stdout, opts)
	case LogFormatJSON:
		baseHandler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		return fmt.Errorf("%w: %s", errInvalidLogFormat, cfg.Format)
	}

	slog.SetDefault(slog.New(baseHandler))

	return nil
}
