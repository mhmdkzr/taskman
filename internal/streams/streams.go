// Package streams aggregates and creates all JetStream streams.
package streams

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/mhmdkzr/taskman/internal/publisher"
)

// CreateStreams creates or updates all required JetStream streams by calling
// each module's CreateStreams function.
func CreateStreams(ctx context.Context, js jetstream.JetStream) error {
	for _, fn := range []func(context.Context, jetstream.JetStream) error{
		publisher.CreateStreams,
	} {
		if err := fn(ctx, js); err != nil {
			return err
		}
	}

	return nil
}
