package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

// Produce publishes an event to the given JetStream subject with idempotent deduplication via MsgID().
func Produce[Event interface{ MsgID() string }](
	ctx context.Context,
	js jetstream.JetStream,
	subject string,
	event Event,
) error {
	if js == nil {
		return fmt.Errorf("nil jetstream client")
	}
	if subject == "" {
		return fmt.Errorf("empty subject")
	}
	if event.MsgID() == "" {
		return fmt.Errorf("empty message id")
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if _, err := js.Publish(ctx, subject, payload, jetstream.WithMsgID(event.MsgID())); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
