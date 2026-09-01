package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/web/components"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	cols, err := s.board(r)
	if err != nil {
		http.Error(w, "list tasks: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := components.Page(cols).Render(r.Context(), w); err != nil {
		slog.Error("web: index render", "error", err)
	}
}

// handleTaskList serves the #task-list kanban fragment, used both by the
// periodic refresh and by the live /events stream.
func (s *Server) handleTaskList(w http.ResponseWriter, r *http.Request) {
	cols, err := s.board(r)
	if err != nil {
		http.Error(w, "list tasks: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := components.Board(cols).Render(r.Context(), w); err != nil {
		slog.Error("web: task list render", "error", err)
	}
}

// handleTaskDetail serves the #detail fragment for one task. The UI fetches
// it when a task is selected and re-fetches it on an interval while selected,
// so a running task's activity stays live.
func (s *Server) handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	row, err := s.taskDetail(r, id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := components.TaskDrawer(row).Render(r.Context(), w); err != nil {
		slog.Error("web: task detail render", "error", err)
	}
}

// board loads all tasks and groups them into one kanban column per status, in
// lifecycle order, each card carrying its live phase.
func (s *Server) board(r *http.Request) ([]components.KanbanCol, error) {
	tasks, err := task.ListTasks(r.Context(), s.store.RO(), "")
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	order := []task.TaskStatus{
		task.TaskStatusCreated,
		task.TaskStatusStarted,
		task.TaskStatusCompleted,
		task.TaskStatusReviewed,
	}
	labels := map[task.TaskStatus]string{
		task.TaskStatusCreated:   "Created",
		task.TaskStatusStarted:   "Started",
		task.TaskStatusCompleted: "Completed",
		task.TaskStatusReviewed:  "Reviewed",
	}
	buckets := make(map[task.TaskStatus][]components.TaskRow, len(order))
	for _, t := range tasks {
		st := t.Status
		if _, ok := labels[st]; !ok {
			continue // defensive: unknown statuses are not rendered
		}
		buckets[st] = append(buckets[st], components.TaskRow{Task: t, Phase: s.phase(t.ID.String())})
	}

	cols := make([]components.KanbanCol, 0, len(order))
	for _, st := range order {
		cols = append(cols, components.KanbanCol{Status: string(st), Label: labels[st], Tasks: buckets[st]})
	}
	return cols, nil
}

// taskDetail builds the detail drawer's data for one task.
func (s *Server) taskDetail(r *http.Request, id string) (components.TaskDetail, error) {
	t, err := s.getTask(r, id)
	if err != nil {
		return components.TaskDetail{}, err
	}
	return components.TaskDetail{
		Task:     *t,
		Phase:    s.phase(id),
		Sessions: s.renderSessions(r, t),
	}, nil
}

// jsonError is a minimal JSON error helper (kept in web, since the fragment
// handlers are the only JSON-free ones in the package).
func jsonError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
		slog.Error("web: write error response", "error", err)
	}
}

// getTask loads one task by id.
func (s *Server) getTask(r *http.Request, id string) (*task.Task, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid task id %q: %w", id, err)
	}
	t, err := task.GetTask(r.Context(), s.store.RO(), parsed)
	if err != nil {
		return nil, fmt.Errorf("get task %s: %w", id, err)
	}
	return t, nil
}

// renderSessions renders the activity for a task's execution and review
// session transcripts, each oldest turn first.
func (s *Server) renderSessions(r *http.Request, t *task.Task) []components.SessionActivity {
	sessions := []struct {
		id    uuid.UUID
		label string
	}{
		{t.SessionID, "execution"},
		{t.ReviewSessionID, "review"},
	}
	var out []components.SessionActivity
	for _, sess := range sessions {
		if sess.id == uuid.Nil() {
			continue
		}
		tr, err := store.GetSessionTranscript(r.Context(), s.store.RO(), sess.id.String())
		if err != nil {
			continue
		}
		activity := components.SessionActivity{Label: sess.label}
		for _, turn := range tr.Turns {
			activity.Turns = append(activity.Turns, activityFromTurn(turn))
		}
		if len(activity.Turns) > 0 {
			out = append(out, activity)
		}
	}
	return out
}

// activityFromTurn flattens one transcript turn into renderable activity.
func activityFromTurn(turn store.TurnTranscript) components.ActivityTurn {
	out := components.ActivityTurn{Prompt: turn.Prompt.Text()}
	for _, step := range turn.Steps {
		st := components.ActivityStep{}
		// results are keyed by tool call id so a result lands on its call.
		byCall := map[string]*components.ActivityTool{}
		for _, m := range step.Messages {
			for _, p := range m.Parts {
				switch p.Type {
				case "reasoning":
					st.Reasoning += p.Text
				case "text":
					st.Text += p.Text
				case "tool-call":
					st.Tools = append(st.Tools, components.ActivityTool{Name: p.ToolName, Input: p.ToolInput})
					byCall[p.ToolCallID] = &st.Tools[len(st.Tools)-1]
				case "tool-result":
					if tool, ok := byCall[p.ToolCallID]; ok {
						tool.Output = p.ToolOutput
						tool.Error = p.ToolError
					}
				}
			}
		}
		out.Steps = append(out.Steps, st)
	}
	return out
}
