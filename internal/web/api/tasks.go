package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"

	"github.com/starfederation/datastar-go/datastar"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools/task"
	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/web/components"
	"github.com/mhmdkzr/loop/pkg/jsonresp"
)

// httpStatusForError maps a domain error to its HTTP status; anything
// unrecognized is a 500, per CLAUDE.md's error-handling conventions.
func httpStatusForError(err error) int {
	switch {
	case errors.Is(err, sessions.ErrSessionNotFound), errors.Is(err, task.ErrTaskNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// indexHandler serves the entire UI at "/": every task's full detail, in
// one page. A normal navigation (no "datastar" query param) renders the full
// HTML document; the page's own ambient poll (see components.refreshAction)
// hits this same route with that param set, so a plain GET and a Datastar
// action share one handler instead of needing a second endpoint just for
// the periodic refresh.
func indexHandler(a app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		view, err := buildTasksView(r.Context(), a)
		if err != nil {
			jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
			return
		}
		if r.URL.Query().Has(datastar.DatastarKey) {
			if err := datastar.NewSSE(w, r).PatchElementTempl(components.App(view)); err != nil {
				slog.Error("patch app view", "error", err)
			}
			return
		}
		if err := components.App(view).Render(r.Context(), w); err != nil {
			slog.Error("render tasks page", "error", err)
		}
	}
}

// buildTasksView loads every task's full detail - specification, planning,
// and every session dispatched for it with that session's full turn
// history - since each task's card expands to show all of it in place
// rather than linking out to a separate page.
func buildTasksView(ctx context.Context, a app.App) (components.AppView, error) {
	tasks, err := task.ListTasks(ctx, a.Deps.Store.RO(), task.TaskFilter{})
	if err != nil {
		return components.AppView{}, fmt.Errorf("list tasks: %w", err)
	}

	// tasks is oldest-first (see task.ListTasks), and the list keeps that
	// order - within each state section (see groupTasksByState), the oldest
	// task reads at the top.
	details := make([]components.TaskDetailView, len(tasks))
	for i, t := range tasks {
		detail, err := buildTaskDetail(ctx, a, t)
		if err != nil {
			return components.AppView{}, err
		}
		details[i] = detail
	}

	return components.AppView{Tasks: details}, nil
}

// buildTaskDetail loads every session linked to t (see task.SessionIDsForTask)
// plus each one's full turn history, tagging each session root or child by
// whether it has a parent session.
func buildTaskDetail(ctx context.Context, a app.App, t task.Task) (components.TaskDetailView, error) {
	sessionIDs, err := task.SessionIDsForTask(ctx, a.Deps.Store.RO(), t.ID)
	if err != nil {
		return components.TaskDetailView{}, fmt.Errorf("get task session ids: %w", err)
	}
	ids := make([]sessions.SessionID, len(sessionIDs))
	for i, sid := range sessionIDs {
		ids[i] = sessions.SessionID(sid)
	}
	summaries, err := sessions.SessionSummariesByIDs(ctx, a.Deps.Store, ids)
	if err != nil {
		return components.TaskDetailView{}, fmt.Errorf("get task sessions: %w", err)
	}
	// SessionSummariesByIDs doesn't order its results (it queries by an IN
	// list), so sort oldest first here - the order a session list should
	// read in.
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].CreatedAt < summaries[j].CreatedAt
	})

	rows := make([]components.TaskSessionRow, len(summaries))
	for i, s := range summaries {
		detail, err := sessions.GetSessionDetail(ctx, a.Deps.Store, s.SessionID)
		if err != nil {
			return components.TaskDetailView{}, fmt.Errorf("get session detail: %w", err)
		}
		rows[i] = components.TaskSessionRow{Session: s, Detail: detail}
	}
	return components.TaskDetailView{Task: t, Sessions: rows}, nil
}
