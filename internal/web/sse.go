package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

// eventsKeepAlive is how often the stream re-renders the task list even when
// the bus is quiet, so CLI-driven changes (add/rm) still show up and the
// connection stays alive.
const eventsKeepAlive = 5 * time.Second

// handleEvents is the live Datastar SSE stream. It subscribes to the bus and,
// for every task-relevant event, re-renders the #task-list fragment and
// patches it into the DOM. The task list is the single live-updating region;
// the detail pane is fetched on demand via /api/tasks/{id}.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	msgs := make(chan sseMsg, 256)
	var unsubs []func()
	for _, subject := range []string{"agent.>", "scheduler.>"} {
		unsub, err := s.pub.SubscribeCore(subject, func(subject string, data []byte) {
			select {
			case msgs <- sseMsg{subject: subject, data: data}:
			default:
				slog.Warn("web: sse client too slow, dropping event", "subject", subject)
			}
		})
		if err != nil {
			for _, u := range unsubs {
				u()
			}
			http.Error(w, fmt.Sprintf("subscribe %s: %v", subject, err), http.StatusInternalServerError)
			return
		}
		unsubs = append(unsubs, unsub)
	}
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	sse := datastar.NewSSE(w, r)

	// Patch the connection indicator immediately, then keep the list fresh.
	if err := s.patchTaskList(sse, r); err != nil {
		return
	}

	ticker := time.NewTicker(eventsKeepAlive)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.patchTaskList(sse, r); err != nil {
				return
			}
		case m := <-msgs:
			s.handleEvent(m)
			if err := s.patchTaskList(sse, r); err != nil {
				return
			}
		}
	}
}

// patchTaskList re-renders the #task-list kanban fragment from the store and
// patches it into the DOM.
func (s *Server) patchTaskList(sse *datastar.ServerSentEventGenerator, r *http.Request) error {
	cols, err := s.board(r)
	if err != nil {
		slog.Error("web: task list render", "error", err)
		return fmt.Errorf("task rows: %w", err)
	}
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "taskList", map[string]any{"Cols": cols}); err != nil {
		slog.Error("web: task list render", "error", err)
		return fmt.Errorf("render task list: %w", err)
	}
	if err := sse.PatchElements(buf.String()); err != nil {
		slog.Error("web: patch task list", "error", err)
		return fmt.Errorf("patch task list: %w", err)
	}
	return nil
}

// handleEvent records the transient live state an event carries so the
// fragments can render it: the current pipeline phase per task.
func (s *Server) handleEvent(m sseMsg) {
	if m.subject != "agent.pipeline.phase" {
		return
	}
	var ev struct {
		TaskID string `json:"task_id"`
		Phase  string `json:"phase"`
	}
	if err := json.Unmarshal(m.data, &ev); err != nil || ev.TaskID == "" {
		return
	}
	s.setPhase(ev.TaskID, ev.Phase)
}

// sseMsg is one bus message ready to be handled by the live stream.
type sseMsg struct {
	subject string
	data    []byte
}
