package web

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

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
	return &Server{store: st, pub: pub, phases: map[string]string{}}, st, pub
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

func (s *Server) routesHandler() http.Handler {
	mux := http.NewServeMux()
	for _, r := range s.Routes() {
		mux.HandleFunc(r.Method+" "+r.Path, r.Handler)
	}
	return mux
}

func TestIndexRendersTaskList(t *testing.T) {
	srv, st, _ := newTestServer(t)
	insertTestTask(t, st.RW())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.routesHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Fix nil pointer", "data-signals", "datastar.js", "id=\"task-list\"", "Created", "Started", "Completed", "Reviewed"} {
		if !strings.Contains(body, want) {
			t.Errorf("index missing %q", want)
		}
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("index content type = %q, want text/html", ct)
	}
}

// TestNoCommandRoutes verifies the dashboard is read-only: there is no route
// that would trigger a run from the browser.
func TestNoCommandRoutes(t *testing.T) {
	srv, _, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/commands", strings.NewReader(`{"task_ids":["abc"]}`))
	rec := httptest.NewRecorder()
	srv.routesHandler().ServeHTTP(rec, req)
	if rec.Code == http.StatusOK || rec.Code == http.StatusAccepted {
		t.Fatalf("POST /api/commands = %d, want non-2xx (read-only dashboard)", rec.Code)
	}
}

// TestTaskListFragmentRendersTasks verifies /api/tasks returns the kanban
// fragment with the inserted task grouped under its status column.
func TestTaskListFragmentRendersTasks(t *testing.T) {
	srv, st, _ := newTestServer(t)
	insertTestTask(t, st.RW())

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rec := httptest.NewRecorder()
	srv.routesHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/tasks = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Fix nil pointer", "Created", "Started", "Completed", "Reviewed"} {
		if !strings.Contains(body, want) {
			t.Errorf("task list fragment missing %q", want)
		}
	}
	// The created task must be in the Created column: the "Created" heading
	// comes before the task title in the fragment.
	if !strings.Contains(body, "Created") || !strings.Contains(body, "Fix nil pointer") {
		t.Fatalf("task list fragment malformed:\n%s", body)
	}
	if idx := strings.Index(body, "Created"); idx < 0 || strings.Index(body, "Fix nil pointer") < idx {
		t.Errorf("task title should follow the Created column heading:\n%s", body)
	}
}

// TestTaskDetailFragment verifies /api/tasks/{id} renders the detail fragment
// for a task.
func TestTaskDetailFragment(t *testing.T) {
	srv, st, _ := newTestServer(t)
	tk := insertTestTask(t, st.RW())

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/"+tk.ID.String(), nil)
	rec := httptest.NewRecorder()
	srv.routesHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/tasks/{id} = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"id=\"detail\"", "Fix nil pointer", "Guard against a nil client"} {
		if !strings.Contains(body, want) {
			t.Errorf("detail fragment missing %q", want)
		}
	}
}

// TestTaskDetailRendersAgentMessages verifies the activity rendering shows the
// agent's reasoning, tool calls/results and assistant text from the session
// transcript.
func TestTaskDetailRendersAgentMessages(t *testing.T) {
	srv, st, _ := newTestServer(t)

	// Build a task with a real execution session carrying one turn with a
	// reasoning + tool-call step and a final text step.
	sessionID, err := store.NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	callID := "call_01"
	if err := store.SaveRun(context.Background(), st.RW(), sessionID, store.Run{
		Model:           "test-model",
		ReasoningEffort: "medium",
		MaxSteps:        100,
		SystemPrompt:    "You are an agent.",
		Prompt:          "Fix the nil pointer.",
		Messages: []store.Message{
			{
				Role: "assistant", Number: 1, FinishReason: "tool-calls",
				Parts: []store.Part{
					{Type: "reasoning", Text: "I'll guard the client."},
					{
						Type:       "tool-call",
						ToolCallID: callID,
						ToolName:   "edit",
						ToolInput:  `{"path":"send.go","old":"","new":"x"}`,
					},
				},
			},
			{
				Role: "tool", Number: 1,
				Parts: []store.Part{{
					Type: "tool-result", ToolCallID: callID, ToolName: "edit",
					ToolOutput: `{"bytes_written":1}`,
				}},
			},
			{
				Role: "assistant", Number: 2, FinishReason: "stop",
				Parts: []store.Part{{Type: "text", Text: "Done — added the nil check."}},
			},
		},
	}); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		t.Fatalf("parse session id: %v", err)
	}

	tk := insertTestTask(t, st.RW())
	if _, err := st.RW().Exec(
		`UPDATE tasks SET session_id = ? WHERE id = ?`, sid.String(), tk.ID.String(),
	); err != nil {
		t.Fatalf("link session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/"+tk.ID.String(), nil)
	rec := httptest.NewRecorder()
	srv.routesHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/tasks/{id} = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"execution", "Turn 1", "I&#39;ll guard the client.", "edit", "Done — added the nil check."} {
		if !strings.Contains(body, want) {
			t.Errorf("detail activity missing %q", want)
		}
	}
	if !strings.Contains(body, `&#34;bytes_written&#34;:1`) {
		t.Errorf("detail activity missing tool result output")
	}
}

// TestEventsStreamPatchesLiveEvents verifies /events is a Datastar SSE stream:
// it emits a datastar-patch-elements event carrying the task list after a
// pipeline phase is published.
func TestEventsStreamPatchesLiveEvents(t *testing.T) {
	srv, _, pub := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.routesHandler().ServeHTTP(rec, req)
	}()

	time.Sleep(150 * time.Millisecond)
	_ = pub.Publish(context.Background(), events.PipelinePhase{
		TaskID: "abc", Phase: "execution", Timestamp: time.Now().UTC(),
	})

	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("no datastar patch received on SSE stream")
		default:
			body := rec.Body.String()
			if strings.Contains(body, "datastar-patch-elements") {
				cancel()
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}
