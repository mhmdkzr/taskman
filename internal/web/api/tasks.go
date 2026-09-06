package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"uuid"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools/task"
	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/web/components"
	"github.com/mhmdkzr/loop/pkg/jsonresp"
)

// tasksPageHandler renders the app shell in Tasks mode with the most
// recently created task active, or no task active if none exist yet.
func tasksPageHandler(a app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		view, err := buildTasksView(r.Context(), a, nil)
		if err != nil {
			jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
			return
		}
		if len(view.Tasks) > 0 {
			id := view.Tasks[0].ID
			detail, err := buildTaskDetail(r.Context(), a, id)
			if err != nil {
				jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
				return
			}
			view.ActiveTask = detail
		}
		if err := components.App(view).Render(r.Context(), w); err != nil {
			slog.Error("render tasks page", "error", err)
		}
	}
}

// taskPageHandler renders the app shell in Tasks mode with one task active,
// for direct navigation (a bookmarked or shared task URL).
func taskPageHandler(a app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseTaskID(r.PathValue("id"))
		if err != nil {
			jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
			return
		}
		view, err := buildTasksView(r.Context(), a, &id)
		if err != nil {
			jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
			return
		}
		if err := components.App(view).Render(r.Context(), w); err != nil {
			slog.Error("render task page", "error", err)
		}
	}
}

func parseTaskID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("parse task id: %w", err)
	}
	return id, nil
}

// buildTasksView loads every task for the rail, newest first, plus - when
// activeID is non-nil - that task's full detail for the main pane.
func buildTasksView(ctx context.Context, a app.App, activeID *uuid.UUID) (components.AppView, error) {
	tasks, err := task.ListTasks(ctx, a.Deps.Store.RO(), task.TaskFilter{})
	if err != nil {
		return components.AppView{}, fmt.Errorf("list tasks: %w", err)
	}

	summaries := make([]components.TaskSummary, len(tasks))
	for i, t := range tasks {
		sessionIDs, err := task.SessionIDsForTask(ctx, a.Deps.Store.RO(), t.ID)
		if err != nil {
			return components.AppView{}, fmt.Errorf("count task sessions: %w", err)
		}
		// tasks is oldest-first (see task.ListTasks); the rail shows newest
		// first, matching the session list.
		summaries[len(tasks)-1-i] = components.TaskSummary{Task: t, SessionCount: len(sessionIDs)}
	}

	view := components.AppView{Mode: "tasks", Tasks: summaries}
	if activeID == nil {
		return view, nil
	}
	detail, err := buildTaskDetail(ctx, a, *activeID)
	if err != nil {
		return components.AppView{}, err
	}
	view.ActiveTask = detail
	return view, nil
}

// buildTaskDetail loads one task plus every session linked to it (see
// task.SessionIDsForTask), each tagged root or child by whether it has a
// parent session.
func buildTaskDetail(ctx context.Context, a app.App, id uuid.UUID) (*components.TaskDetailView, error) {
	t, err := task.GetTask(ctx, a.Deps.Store.RO(), id)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	sessionIDs, err := task.SessionIDsForTask(ctx, a.Deps.Store.RO(), id)
	if err != nil {
		return nil, fmt.Errorf("get task session ids: %w", err)
	}
	ids := make([]sessions.SessionID, len(sessionIDs))
	for i, sid := range sessionIDs {
		ids[i] = sessions.SessionID(sid)
	}
	summaries, err := sessions.SessionSummariesByIDs(ctx, a.Deps.Store, ids)
	if err != nil {
		return nil, fmt.Errorf("get task sessions: %w", err)
	}

	rows := make([]components.TaskSessionRow, len(summaries))
	for i, s := range summaries {
		role := "root"
		if s.ParentSessionID != nil {
			role = "child"
		}
		rows[i] = components.TaskSessionRow{Session: s, Role: role}
	}
	return &components.TaskDetailView{Task: t, Sessions: rows}, nil
}
