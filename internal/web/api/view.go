package api

import (
	"context"
	"fmt"
	"sort"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools/task"
	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/web/components"
)

func tasksView(ctx context.Context, a app.App) (components.AppView, error) {
	tasks, err := task.ListTasks(ctx, a.Deps.Store.RO(), task.TaskFilter{})
	if err != nil {
		return components.AppView{}, fmt.Errorf("list tasks: %w", err)
	}

	details := make([]components.TaskDetailView, len(tasks))
	for i, t := range tasks {
		detail, err := taskDetailView(ctx, a, t)
		if err != nil {
			return components.AppView{}, err
		}
		details[i] = detail
	}

	return components.AppView{Tasks: details}, nil
}

func taskDetailView(ctx context.Context, a app.App, t task.Task) (components.TaskDetailView, error) {
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
