package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/task"
)

//go:embed index.html task_list.html task_detail.html
var files embed.FS

// templates is parsed once: index.html is the page shell, taskList and
// taskDetail are fragments the server renders into /api/tasks and
// /api/tasks/{id} (and which /events patches live).
var templates = template.Must(template.New("index.html").ParseFS(files, "*.html"))

// taskRow is one entry in the task list fragment: a task plus its live phase.
type taskRow struct {
	task.Task

	Phase string
}

// taskDetailData renders the detail fragment for one task.
type taskDetailData struct {
	Task     task.Task
	Phase    string
	Activity []activityEntry
	Diff     []diffFile
}

// activityEntry is one turn of a session transcript, rendered in the detail.
type activityEntry struct {
	Turn   int
	Prompt string
	Lines  []activityLine
}

// activityLine is one line of activity within a turn: a tool call, a tool
// error, or an assistant text snippet.
type activityLine struct {
	Kind string // "tool" | "err" | "text"
	Text string
}

// diffFile is one file's change in a task's diff, rendered in the detail.
type diffFile struct {
	Name       string
	ChangeType string
	Additions  int
	Deletions  int
	Lines      []diffLine
}

// diffLine is one line of a unified diff patch with its render class.
type diffLine struct {
	Class string // "" | "hunk" | "add" | "del"
	Text  string
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	rows, err := s.taskRows(r)
	if err != nil {
		http.Error(w, "list tasks: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "index.html", map[string]any{"Tasks": rows}); err != nil {
		slog.Error("web: index render", "error", err)
	}
}

// handleTaskList serves the #task-list fragment, used both by the periodic
// refresh and by the live /events stream.
func (s *Server) handleTaskList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.taskRows(r)
	if err != nil {
		http.Error(w, "list tasks: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "taskList", map[string]any{"Tasks": rows}); err != nil {
		slog.Error("web: task list render", "error", err)
	}
}

// handleTaskDetail serves the #detail fragment for one task. The UI fetches
// it when a task is selected and re-fetches it on an interval while selected,
// so a running task's activity and diff stay live.
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

// taskRows loads all tasks and merges each with its live phase (tracked from
// agent.pipeline.phase events by the /events stream).
func (s *Server) taskRows(r *http.Request) ([]taskRow, error) {
	tasks, err := task.ListTasks(r.Context(), s.store.RO(), "")
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	rows := make([]taskRow, 0, len(tasks))
	for _, t := range tasks {
		rows = append(rows, taskRow{Task: t, Phase: s.phase(t.ID.String())})
	}
	return rows, nil
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
		Activity: s.renderActivity(r, t),
		Diff:     s.renderDiff(id),
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

// renderActivity builds the structured activity for a task's execution and
// review session transcripts, newest turn first.
func (s *Server) renderActivity(r *http.Request, t *task.Task) []activityEntry {
	var out []activityEntry
	for _, sid := range []uuid.UUID{t.SessionID, t.ReviewSessionID} {
		if sid == uuid.Nil() {
			continue
		}
		tr, err := store.GetSessionTranscript(r.Context(), s.store.RO(), sid.String())
		if err != nil {
			continue
		}
		for i, turn := range slices.Backward(tr.Turns) {
			_ = i
			entry := activityEntry{Turn: turn.Turn.Number, Prompt: turn.Prompt.Text()}
			for _, step := range turn.Steps {
				for _, m := range step.Messages {
					for _, p := range m.Parts {
						switch p.Type {
						case "tool-call":
							entry.Lines = append(entry.Lines, activityLine{
								Kind: "tool",
								Text: shorten(p.ToolInput, 200),
							})
						case "tool-result":
							if p.ToolError != "" {
								entry.Lines = append(entry.Lines, activityLine{
									Kind: "err",
									Text: shorten(p.ToolError, 200),
								})
							}
						case "text":
							entry.Lines = append(entry.Lines, activityLine{
								Kind: "text",
								Text: shorten(p.Text, 300),
							})
						}
					}
				}
			}
			out = append(out, entry)
		}
	}
	return out
}

// renderDiff builds the structured diff for a task.
func (s *Server) renderDiff(taskID string) []diffFile {
	d := s.diffFor(taskID)
	if d == nil {
		return nil
	}
	out := make([]diffFile, 0, len(d.Files))
	for _, f := range d.Files {
		file := diffFile{
			Name:       f.Name,
			ChangeType: f.ChangeType,
			Additions:  f.Additions,
			Deletions:  f.Deletions,
		}
		for line := range strings.SplitSeq(f.Patch, "\n") {
			cls := ""
			switch {
			case strings.HasPrefix(line, "@@"):
				cls = "hunk"
			case strings.HasPrefix(line, "+"):
				cls = "add"
			case strings.HasPrefix(line, "-"):
				cls = "del"
			}
			file.Lines = append(file.Lines, diffLine{Class: cls, Text: line})
		}
		out = append(out, file)
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
