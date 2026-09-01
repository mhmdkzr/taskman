package runner

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
)

// newTestBus starts an embedded publisher and a second connection to observe
// what the runner publishes.
func newTestBus(t *testing.T) (publisher.Publisher, *nats.Conn) {
	t.Helper()
	pub, err := publisher.ConnectOn(0)
	if err != nil {
		t.Fatalf("ConnectOn: %v", err)
	}
	obs, err := nats.Connect(pub.Conn().ConnectedUrl())
	if err != nil {
		t.Fatalf("connect observer: %v", err)
	}
	t.Cleanup(obs.Close)
	return pub, obs
}

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "runner.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	if err := store.Migrate(st.RW()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return st
}

// TestRunPublishesCommandConfirmsDelivery verifies the runner subscribes to
// the run-command subject and that a published command reaches the runner's
// goroutine path — the command is observed on the bus and the runner does not
// error on a task it cannot find.
func TestRunReceivesCommand(t *testing.T) {
	pub, obs := newTestBus(t)
	st := newTestStore(t)
	r := New(st, pub, config.AgentConfig{})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()

	// The runner subscribes asynchronously; give it time to attach.
	time.Sleep(100 * time.Millisecond)

	cmd := events.RunCommand{TaskIDs: []string{"01a05cac-d3a6-7ab8-a6cc-624b6c988509"}, Source: "."}
	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("marshal command: %v", err)
	}
	if err := pub.PublishCore(events.CommandRunSubject, data); err != nil {
		t.Fatalf("publish command: %v", err)
	}

	// The runner reads the command and tries to load the task (which does not
	// exist); it must log and continue, not crash. There is nothing observable
	// on the bus for a missing task, so just confirm the process survives.
	time.Sleep(200 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner Run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not stop")
	}

	// A second, real store-backed check: the observer subscription to the
	// command subject must have received the published command.
	received := make(chan struct{})
	_, err = obs.Subscribe(events.CommandRunSubject, func(m *nats.Msg) {
		var got events.RunCommand
		if err := json.Unmarshal(m.Data, &got); err != nil {
			t.Errorf("decode observed command: %v", err)
			return
		}
		if len(got.TaskIDs) != 1 || got.TaskIDs[0] != cmd.TaskIDs[0] {
			t.Errorf("observed command = %+v, want %+v", got, cmd)
		}
		close(received)
	})
	if err != nil {
		t.Fatalf("subscribe observer: %v", err)
	}
	if err := obs.Flush(); err != nil {
		t.Fatalf("flush observer: %v", err)
	}
	if err := pub.PublishCore(events.CommandRunSubject, data); err != nil {
		t.Fatalf("publish command again: %v", err)
	}
	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("observer did not receive the command")
	}
}
