package publisher

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

// CreateStreams creates the TASKMAN stream that captures every message on the
// agent and scheduler subjects and deduplicates by MsgID within the dedup
// window. It is idempotent: a restart reuses the existing stream as-is.
func CreateStreams(ctx context.Context, js jetstream.JetStream) error {
	_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       StreamName,
		Subjects:   []string{"agent.>", "scheduler.>"},
		Storage:    jetstream.MemoryStorage,
		Duplicates: dedupWindow,
	})
	if err != nil {
		return fmt.Errorf("create stream %s: %w", StreamName, err)
	}
	return nil
}
