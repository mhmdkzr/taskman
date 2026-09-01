package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/task"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(dir + "/test.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	if err := store.Migrate(st.RW()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return st
}

func newTestServer(t *testing.T) (*Server, *store.Store, publisher.Publisher) {
	t.Helper()
	st := newTestStore(t)
	pub, err := publisher.ConnectOn(0)
	if err != nil {
		t.Fatalf("connect publisher: %v", err)
	}
	srv, err := New(Config{Addr: "127.0.0.1:0", Store: st, Pub: pub})
	if err != nil {
		t.Fatalf("new web server: %v", err)
	}
	return srv, st, pub
}

func insertTestTask(t *testing.T, db *sql.DB) task.Task {
	t.Helper()
	tk := task.Task{
		ID:            task.NewTaskID(),
		Title:         "Fix nil pointer",
		TaskType:      "bug",
		What:          "Guard against a nil client before calling Send.",
		Why:           "Send panics when the client is nil.",
		How:           "Add a nil check at the top of Send.",
		Urgency:       "low",
		Importance:    "medium",
		Packages:      []string{"internal/foo"},
		CompletedWhen: []string{"tests pass"},
		Variant:       "medium",
	}
	if err := task.CreateTask(context.Background(), db, tk); err != nil {
		t.Fatalf("create task: %v", err)
	}
	return tk
}

func get(t *testing.T, srv *Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestIndexRendersTasks(t *testing.T) {
	srv, st, _ := newTestServer(t)
	insertTestTask(t, st.RW())

	rec := get(t, srv, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Fix nil pointer") {
		t.Errorf("index does not render task title:\n%s", body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("index is not HTML content type: %s", ct)
	}
}

func TestOverviewReturnsTasks(t *testing.T) {
	srv, st, _ := newTestServer(t)
	tk := insertTestTask(t, st.RW())

	rec := get(t, srv, "/api/overview")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/overview = %d, want 200", rec.Code)
	}
	var ov overview
	if err := json.Unmarshal(rec.Body.Bytes(), &ov); err != nil {
		t.Fatalf("decode overview: %v", err)
	}
	if len(ov.Tasks) != 1 || ov.Tasks[0].ID != tk.ID {
		t.Errorf("overview tasks = %+v, want the inserted task", ov.Tasks)
	}
}

// TestCommandsPublishForwardsToBus verifies POST /api/commands publishes a
// RunCommand on the core command subject, which a subscriber can observe.
func TestCommandsPublishForwardsToBus(t *testing.T) {
	srv, st, pub := newTestServer(t)
	tk := insertTestTask(t, st.RW())

	got := make(chan []byte, 1)
	unsub, err := pub.SubscribeCore(events.CommandRunSubject, func(_ string, data []byte) {
		select {
		case got <- data:
		default:
		}
	})
	if err != nil {
		t.Fatalf("subscribe command subject: %v", err)
	}
	defer unsub()

	body := strings.NewReader(fmt.Sprintf(`{"task_ids":[%q]}`, tk.ID))
	req := httptest.NewRequest(http.MethodPost, "/api/commands", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("POST /api/commands = %d, want 202", rec.Code)
	}

	select {
	case data := <-got:
		var cmd events.RunCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			t.Fatalf("decode command: %v", err)
		}
		if len(cmd.TaskIDs) != 1 || cmd.TaskIDs[0] != tk.ID.String() {
			t.Errorf("command task ids = %v, want [%s]", cmd.TaskIDs, tk.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no command observed on the bus")
	}
}

func TestCommandsRejectsEmpty(t *testing.T) {
	srv, _, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/commands", strings.NewReader(`{"task_ids":[]}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST empty commands = %d, want 400", rec.Code)
	}
}

// TestEventsStreamsLiveEvents verifies /events is an SSE stream that forwards
// a published pipeline event to the client.
func TestEventsStreamsLiveEvents(t *testing.T) {
	srv, _, pub := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.Handler().ServeHTTP(rec, req)
	}()

	// Give the handler time to subscribe, then publish an event.
	time.Sleep(100 * time.Millisecond)
	_ = pub.Publish(context.Background(), events.PipelinePhase{
		TaskID: "abc", Phase: "execution", Timestamp: time.Now().UTC(),
	})

	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("event not received on SSE stream")
		default:
			body := rec.Body.String()
			if strings.Contains(body, "agent.pipeline.phase") {
				if !strings.Contains(body, "execution") {
					t.Errorf("SSE payload missing phase value: %s", body)
				}
				cancel()
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}
