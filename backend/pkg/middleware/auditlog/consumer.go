package auditlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	streamName   = "API_AUDIT"
	consumerName = "audit-log-db-writer"
	subject      = SubjectAPIAudit
)

// Start starts the audit log consumer that writes events to the database.
func Start(ctx context.Context, js jetstream.JetStream, db *sql.DB, timeout time.Duration) error {
	if ctx == nil {
		return fmt.Errorf("nil context")
	}
	if db == nil {
		return fmt.Errorf("nil db")
	}
	if js == nil {
		return fmt.Errorf("nil jetstream client")
	}

	setupCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		setupCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	consumer, err := js.CreateOrUpdateConsumer(setupCtx, streamName, jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		FilterSubject: subject,
	})
	if err != nil {
		return fmt.Errorf("create consumer %s: %w", consumerName, err)
	}

	_, err = consumer.Consume(func(msg jetstream.Msg) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in audit log consumer", "consumer", consumerName, "panic", r)
				if err := msg.Nak(); err != nil {
					slog.Error("failed to nak audit event after panic", "consumer", consumerName, "error", err)
				}
			}
		}()

		var event EventAPIAuditLogged
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			slog.Error("failed to unmarshal audit event", "consumer", consumerName, "error", err)
			if err := msg.Nak(); err != nil {
				slog.Error("failed to nak malformed audit event", "consumer", consumerName, "error", err)
			}
			return
		}

		if err := insertAuditLog(ctx, db, event); err != nil {
			slog.Error("failed to insert audit log", "consumer", consumerName, "error", err)
			if err := msg.Nak(); err != nil {
				slog.Error("failed to nak audit event after insert failure", "consumer", consumerName, "error", err)
			}
			return
		}

		if err := msg.Ack(); err != nil {
			slog.Error("failed to ack audit event", "consumer", consumerName, "error", err)
		}
	})
	if err != nil {
		return fmt.Errorf("start consumer %s: %w", consumerName, err)
	}
	return nil
}
