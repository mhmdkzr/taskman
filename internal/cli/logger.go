package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"
)

// initLogger sets up slog per the root --log-level/--log-format flags,
// before any command runs.
func initLogger(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cmd.String("log-level"))); err != nil {
		return ctx, fmt.Errorf("parse --log-level: %w", err)
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level, AddSource: true}
	switch format := cmd.String("log-format"); format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		return ctx, fmt.Errorf("--log-format must be %q or %q, got %q", "text", "json", format)
	}
	slog.SetDefault(slog.New(handler))
	return ctx, nil
}
