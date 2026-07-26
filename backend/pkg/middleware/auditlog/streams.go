package auditlog

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// duplicateWindow is the deduplication window for JetStream streams.
const duplicateWindow = 24 * time.Hour

const streamAPIAudit = "API_AUDIT"

// CreateStreams creates or updates the JetStream stream for API audit events.
func CreateStreams(ctx context.Context, js jetstream.JetStream) error {
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       streamAPIAudit,
		Subjects:   []string{SubjectAPIAudit},
		Duplicates: duplicateWindow,
	}); err != nil {
		return fmt.Errorf("create stream %s: %w", streamAPIAudit, err)
	}
	return nil
}
