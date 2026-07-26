package logger

import (
	"log/slog"
	"strings"

	temporallog "go.temporal.io/sdk/log"
)

// NewTemporalLogger returns a Temporal SDK logger that preserves the app's
// slog formatting but promotes worker task failure messages to error level.
func NewTemporalLogger(base *slog.Logger) temporallog.Logger {
	if base == nil {
		base = slog.Default()
	}
	return temporalLogger{base: base}
}

// temporalLogger wraps a slog.Logger to adapt it to the Temporal SDK logger interface.
type temporalLogger struct {
	base *slog.Logger
}

// Debug logs at debug level with Temporal-style key-value pairs.
func (l temporalLogger) Debug(msg string, keyvals ...any) {
	l.base.Debug(msg, keyvals...)
}

// Info logs at info level, promoting worker task failure messages to error.
func (l temporalLogger) Info(msg string, keyvals ...any) {
	if shouldPromoteTemporalInfo(msg) {
		l.base.Error(msg, keyvals...)
		return
	}
	l.base.Info(msg, keyvals...)
}

// Warn logs at warn level with Temporal-style key-value pairs.
func (l temporalLogger) Warn(msg string, keyvals ...any) {
	l.base.Warn(msg, keyvals...)
}

// Error logs at error level with Temporal-style key-value pairs.
func (l temporalLogger) Error(msg string, keyvals ...any) {
	l.base.Error(msg, keyvals...)
}

// shouldPromoteTemporalInfo returns true for messages that should be promoted to error level.
func shouldPromoteTemporalInfo(msg string) bool {
	return strings.HasPrefix(msg, "Task processing failed with ")
}
