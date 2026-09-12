// Package md renders a task as a human-readable Markdown document, for
// display outside taskman itself (e.g. in a PR description or a chat
// message) - not a replacement for `get --json`, which remains the
// machine-readable source of truth.
package md

import (
	"bytes"
	"context"
	"fmt"
	tmpl "text/template"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

var tpl = tmpl.Must(tmpl.New("template.md").Parse(template))

// Render reads taskID from st and renders it as Markdown. It renders a task
// at any state: every section a task doesn't yet have (Specification,
// Implementation, Blocked, Abandoned, a Git merge, ...) is simply omitted,
// rather than shown empty.
func Render(ctx context.Context, st *store.Store, taskID uuid.UUID) (string, error) {
	t, err := st.Read(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("render task %s: %w", taskID, err)
	}
	return RenderTask(t)
}

// RenderTask renders an already-loaded task as Markdown.
func RenderTask(t task.Task) (string, error) {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, t); err != nil {
		return "", fmt.Errorf("render task %s: %w", t.ID, err)
	}
	return buf.String(), nil
}
