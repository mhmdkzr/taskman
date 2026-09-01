package web

import (
	_ "embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/pkg/jsonresp"
)

//go:embed index.html
var indexHTML string

// pageData is what the index template renders on first paint: a snapshot of
// the current tasks. Live updates then arrive over /events.
type pageData struct {
	Tasks []task.Task
}

// sessionView is the read-only shape of a session for the overview.
type sessionView struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	Model     string `json:"model"`
	ParentID  string `json:"parent_id,omitempty"`
}

// overview is the initial snapshot the UI loads: all tasks plus recent
// sessions, newest first.
type overview struct {
	Tasks    []task.Task   `json:"tasks"`
	Sessions []sessionView `json:"sessions"`
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	tasks, err := task.ListTasks(r.Context(), s.store.RO(), "")
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusInternalServerError, fmt.Errorf("list tasks: %w", err))
		return
	}
	tmpl, err := template.New("index").Parse(indexHTML)
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusInternalServerError, fmt.Errorf("parse template: %w", err))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, pageData{Tasks: tasks}); err != nil {
		slog.Error("web: index render", "error", err)
	}
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	tasks, err := task.ListTasks(r.Context(), s.store.RO(), "")
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusInternalServerError, fmt.Errorf("list tasks: %w", err))
		return
	}
	sessions, err := store.ListSessions(r.Context(), s.store.RO())
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusInternalServerError, fmt.Errorf("list sessions: %w", err))
		return
	}
	out := make([]sessionView, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, sessionView{
			ID:        s.ID,
			CreatedAt: s.CreatedAt,
			Model:     s.Model,
			ParentID:  s.ParentID,
		})
	}
	jsonresp.WriteJSON(w, http.StatusOK, overview{Tasks: tasks, Sessions: out})
}
