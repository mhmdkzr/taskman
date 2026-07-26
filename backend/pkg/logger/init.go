package logger

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/nats-io/nats.go"
)

// Init initializes the logger with optional NATS publishing for warn and error logs.
// You can pass in nil to disable nats integration.
func Init(cfg Config, nc *nats.Conn) error {
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

	if nc == nil {
		slog.SetDefault(slog.New(baseHandler))
	} else {
		var handler slog.Handler = NewNATSHandler(baseHandler, nc)
		slog.SetDefault(slog.New(handler))
	}

	return nil
}
