package logger

import (
	"fmt"
	"log/slog"
)

// Config holds the logger configuration loaded from environment variables.
type Config struct {
	Format LogFormat  `env:"FORMAT"`
	Level  slog.Level `env:"LEVEL"`
}

// LogFormat is a string type for log output format.
type LogFormat string

const (
	// LogFormatJSON is the JSON log format.
	LogFormatJSON LogFormat = "json"
	// LogFormatText is the text log format.
	LogFormatText LogFormat = "text"
)

// Validate returns an error if the configuration is invalid.
func (c Config) Validate() error {
	switch c.Format {
	case LogFormatJSON, LogFormatText:
	default:
		return fmt.Errorf("FORMAT must be %q or %q, got %q", LogFormatJSON, LogFormatText, c.Format)
	}
	if c.Level < slog.LevelDebug || c.Level > slog.LevelError {
		return fmt.Errorf("LEVEL %d is not a valid slog level", c.Level)
	}
	return nil
}

var _ interface{ Validate() error } = Config{}
