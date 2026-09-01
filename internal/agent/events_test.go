package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/publisher"
)

// recorder collects every event published on the agent subject namespace,
// keyed by subject (last message wins), plus the full ordered subject list.
type recorder struct {
	mu       sync.Mutex
	subjects []string
	msgs     map[string]json.RawMessage
}

// recorderID makes a unique durable consumer name so parallel recorders on
// the same bus do not share state.
var recorderID int

func nextRecorderID() string {
	recorderID++
	return fmt.Sprintf("recorder-%d", recorderID)
}

func startRecorder(t *testing.T) (*recorder, publisher.Publisher) {
	t.Helper()
	pub, err := publisher.ConnectOn(0)
	if err != nil {
		t.Fatalf("publisher.Connect: %v", err)
	}
	r := &recorder{msgs: make(map[string]json.RawMessage)}
	sub, err := pub.Subscribe(context.Background(), "agent.>", nextRecorderID(), func(m publisher.Message) {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.subjects = append(r.subjects, m.Subject())
		r.msgs[m.Subject()] = append(json.RawMessage(nil), m.Data()...)
		_ = m.Ack()
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	t.Cleanup(sub.Stop)
	return r, pub
}

func (r *recorder) waitFor(t *testing.T, subject string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		r.mu.Lock()
		_, ok := r.msgs[subject]
		r.mu.Unlock()
		if ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for %q; recorded: %v", subject, r.subjects)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (r *recorder) subjectsSnapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.subjects...)
}

func (r *recorder) decode(t *testing.T, subject string, out any) {
	t.Helper()
	r.mu.Lock()
	data, ok := r.msgs[subject]
	r.mu.Unlock()
	if !ok {
		t.Fatalf("no message recorded for %q", subject)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("unmarshal %q: %v", subject, err)
	}
}
