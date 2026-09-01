package publisher

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/events"
)

func TestPublishErrorsWhenZero(t *testing.T) {
	var p Publisher
	if err := p.Publish(t.Context(), events.TurnStarted{Prompt: "hi"}); err == nil {
		t.Fatal("zero Publisher must error, got nil")
	}
	if _, err := p.Subscribe(t.Context(), "agent.run", "test", func(Message) {}); err == nil {
		t.Fatal("zero Publisher Subscribe must error, got nil")
	}
}

// TestConnectPortBusy verifies that trying to start the embedded bus on a port
// already in use yields a clear error naming the port, not a generic timeout.
func TestConnectPortBusy(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port

	_, err = ConnectOn(port)
	if err == nil {
		t.Fatalf("ConnectOn(%d): want error, got nil", port)
	}
	if !strings.Contains(err.Error(), strconv.Itoa(port)) {
		t.Errorf("error does not name the busy port %d: %v", port, err)
	}
	if !strings.Contains(err.Error(), "not available") {
		t.Errorf("error is not clearly about the port: %v", err)
	}
}

// TestPublishDeliversJSON verifies the real path: Connect starts the embedded
// bus, Publish marshals to JSON, and a subscriber receives it on the event's
// subject.
func TestPublishDeliversJSON(t *testing.T) {
	pub, err := ConnectOn(0)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	got := make(chan Message, 1)
	sub, err := pub.Subscribe(t.Context(), "agent.turn.started", "deliver-test", func(m Message) {
		got <- m
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Stop()

	ev := events.TurnStarted{Prompt: "hello"}
	if err := pub.Publish(t.Context(), ev); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case m := <-got:
		if m.Subject() != "agent.turn.started" {
			t.Errorf("subject = %q", m.Subject())
		}
		var out events.TurnStarted
		if err := json.Unmarshal(m.Data(), &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out.Prompt != "hello" {
			t.Errorf("payload = %+v", out)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no message received")
	}
}

// unmarshalable is an event whose payload cannot be JSON-encoded, exercising
// the marshal-error branch of Publish.
type unmarshalable struct{ F func() }

func (unmarshalable) Subject() string { return "agent.test.bad" }
func (unmarshalable) MsgID() string   { return "unmarshalable" }

// TestPublishDeduplicates verifies exactly-once delivery: re-publishing the
// same event carries the same MsgID, so the stream stores it once and the
// subscriber sees it once.
func TestPublishDeduplicates(t *testing.T) {
	pub, err := ConnectOn(0)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	got := make(chan Message, 2)
	sub, err := pub.Subscribe(t.Context(), "agent.turn.started", "dedup-test", func(m Message) {
		got <- m
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Stop()

	ev := events.TurnStarted{Prompt: "hello"}
	for i := 0; i < 2; i++ {
		if err := pub.Publish(t.Context(), ev); err != nil {
			t.Fatalf("Publish %d: %v", i, err)
		}
	}

	select {
	case m := <-got:
		if len(m.Data()) == 0 {
			t.Error("empty message data")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no message received")
	}
	select {
	case m := <-got:
		t.Errorf("duplicate delivered: %s", m.Data())
	case <-time.After(300 * time.Millisecond):
	}
}

func TestPublishMarshalError(t *testing.T) {
	pub, err := ConnectOn(0)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if err := pub.Publish(t.Context(), unmarshalable{}); err == nil {
		t.Error("Publish of unmarshalable event: want error")
	}
}
