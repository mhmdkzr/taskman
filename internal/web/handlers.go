package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/task"
)

//go:embed index.html task_list.html task_detail.html
var files embed.FS

// templates is parsed once: index.html is the page shell, taskList and
// taskDetail are fragments the server renders into /api/tasks and
// /api/tasks/{id} (and which /events patches live).
var templates = template.Must(template.New("index.html").Funcs(template.FuncMap{
	"shortID":   func(s fmt.Stringer) string { return shorten(s.String(), 13) },
	"shortHash": func(s string) string { return shorten(s, 8) },
}).ParseFS(files, "*.html"))

// kanbanCol is one status column of the board: a heading plus its tasks.
type kanbanCol struct {
	Status string
	Label  string
	Tasks  []taskRow
}

// taskRow is one card in the board: a task plus its live phase.
type taskRow struct {
	task.Task

	Phase string
}

// taskDetailData renders the detail fragment for one task.
type taskDetailData struct {
	Task     task.Task
	Phase    string
	Sessions []sessionActivity
}

// sessionActivity is the rendered transcript of one session (execution or
// review), oldest turn first.
type sessionActivity struct {
	Label string // "execution" | "review"
	Turns []activityTurn
}

// activityTurn is one turn of a session: its prompt and the steps that follow.
type activityTurn struct {
	Number int
	Prompt string
	Steps  []activityStep
}

// activityStep is one tool-loop step within a turn.
type activityStep struct {
	Number    int
	Reasoning string
	Text      string
	Tools     []activityTool
}

// activityTool is one tool call in a step with its result.
type activityTool struct {
	Name   string
	Input  string
	Output string
	Error  string
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	cols, err := s.board(r)
	if err != nil {
		http.Error(w, "list tasks: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "index.html", map[string]any{"Cols": cols}); err != nil {
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
	if err := templates.ExecuteTemplate(w, "taskList", map[string]any{"Cols": cols}); err != nil {
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
	if err := templates.ExecuteTemplate(w, "taskDetail", row); err != nil {
		slog.Error("web: task detail render", "error", err)
	}
}

// board loads all tasks and groups them into one kanban column per status, in
// lifecycle order, each card carrying its live phase.
func (s *Server) board(r *http.Request) ([]kanbanCol, error) {
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
	buckets := make(map[task.TaskStatus][]taskRow, len(order))
	for _, t := range tasks {
		st := t.Status
		if _, ok := labels[st]; !ok {
			continue // defensive: unknown statuses are not rendered
		}
		buckets[st] = append(buckets[st], taskRow{Task: t, Phase: s.phase(t.ID.String())})
	}

	cols := make([]kanbanCol, 0, len(order))
	for _, st := range order {
		cols = append(cols, kanbanCol{Status: string(st), Label: labels[st], Tasks: buckets[st]})
	}
	return cols, nil
}

// taskDetail builds the detail fragment for one task.
func (s *Server) taskDetail(r *http.Request, id string) (*taskDetailData, error) {
	t, err := s.getTask(r, id)
	if err != nil {
		return nil, err
	}
	return &taskDetailData{
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
func (s *Server) renderSessions(r *http.Request, t *task.Task) []sessionActivity {
	sessions := []struct {
		id    uuid.UUID
		label string
	}{
		{t.SessionID, "execution"},
		{t.ReviewSessionID, "review"},
	}
	var out []sessionActivity
	for _, sess := range sessions {
		if sess.id == uuid.Nil() {
			continue
		}
		tr, err := store.GetSessionTranscript(r.Context(), s.store.RO(), sess.id.String())
		if err != nil {
			continue
		}
		activity := sessionActivity{Label: sess.label}
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
func activityFromTurn(turn store.TurnTranscript) activityTurn {
	out := activityTurn{Number: turn.Turn.Number, Prompt: turn.Prompt.Text()}
	for _, step := range turn.Steps {
		st := activityStep{Number: step.Step.Number}
		// results are keyed by tool call id so a result lands on its call.
		byCall := map[string]*activityTool{}
		for _, m := range step.Messages {
			for _, p := range m.Parts {
				switch p.Type {
				case "reasoning":
					st.Reasoning += p.Text
				case "text":
					st.Text += p.Text
				case "tool-call":
					st.Tools = append(st.Tools, activityTool{Name: p.ToolName, Input: p.ToolInput})
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

// shorten truncates s to at most n runes.
func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
