package auditlog

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/mhmdkzr/app/pkg/embeddednats"
	coretestdb "github.com/mhmdkzr/app/pkg/testdb"
	"github.com/mhmdkzr/app/pkg/testenv"
)

func TestStart_ConsumesAuditStreamAndPersistsEvent(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	db := coretestdb.SetupTestPostgres(t, "audit_log_consumer")

	nc, js, err := embeddednats.Connect()
	if err != nil {
		t.Fatalf("connect embedded nats: %v", err)
	}
	t.Cleanup(nc.Close)

	ctx := t.Context()
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
	}); err != nil {
		t.Fatalf("create audit stream: %v", err)
	}

	if err := Start(ctx, js, db, time.Second); err != nil {
		t.Fatalf("start audit consumer: %v", err)
	}

	event := testAuditEvent("req-consumer")
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	if _, err := js.Publish(ctx, subject, payload, jetstream.WithMsgID(event.MsgID())); err != nil {
		t.Fatalf("publish audit event: %v", err)
	}

	waitUntil(t, func() bool {
		var count int
		if err := db.QueryRowContext(context.Background(), `SELECT COUNT(1) FROM core.audit_logs WHERE request_id = $1`, event.RequestID).
			Scan(&count); err != nil {
			return false
		}
		return count == 1
	})
}

func waitUntil(t *testing.T, ok func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
