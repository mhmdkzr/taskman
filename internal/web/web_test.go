package web

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// TestTasksStreamSendsCurrentListOnConnect pins the guarantee the web UI
// relies on: a client that opens (or reopens) the stream is handed the current
// list immediately, rather than only on the next change - otherwise a client
// reconnecting after a missed update would stay stale.
func TestTasksStreamSendsCurrentListOnConnect(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Fatalf("store.Close() error = %v", err)
		}
	})

	def := task.TaskDefinition{Title: "Sync me", Description: "sync the list"}
	if _, err := st.Create(t.Context(), uuid.NewV7(), def, time.Now()); err != nil {
		t.Fatalf("store.Create() error = %v", err)
	}

	srv := httptest.NewServer(NewServer(st).Handler())
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/tasks", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /tasks error = %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })

	type result struct {
		event string
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		event, err := readSSEEvent(bufio.NewReader(resp.Body))
		ch <- result{event: event, err: err}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			t.Fatalf("readSSEEvent() error = %v", res.err)
		}
		if !strings.Contains(res.event, "event: datastar-patch-elements") {
			t.Fatalf("first event = %q, want a datastar-patch-elements event", res.event)
		}
		for _, want := range []string{"#tasks", "Sync me", "Specify"} {
			if !strings.Contains(res.event, want) {
				t.Errorf("first event missing %q; event = %q", want, res.event)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the stream's initial task list event")
	}
}

// readSSEEvent reads a single server-sent event, terminated by a blank line.
func readSSEEvent(r *bufio.Reader) (string, error) {
	var b strings.Builder
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return b.String(), err
		}
		b.WriteString(line)
		if line == "\n" {
			return b.String(), nil
		}
	}
}
