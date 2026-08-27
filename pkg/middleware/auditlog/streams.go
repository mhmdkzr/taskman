package auditlog

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// duplicateWindow is the deduplication window for JetStream streams.
const duplicateWindow = 24 * time.Hour

// CreateStream creates or updates the configured JetStream stream for API audit events.
func CreateStream(ctx context.Context, js jetstream.JetStream, cfg Config) error {
	if ctx == nil {
		return fmt.Errorf("nil context")
	}
	if js == nil {
		return fmt.Errorf("nil jetstream client")
	}
	if err := cfg.validate(); err != nil {
		return err
	}
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       cfg.Stream,
		Subjects:   []string{cfg.Subject},
		Duplicates: duplicateWindow,
	}); err != nil {
		return fmt.Errorf("create stream %s: %w", cfg.Stream, err)
	}
	return nil
}
