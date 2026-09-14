// Package list owns the "list" command: it enumerates tasks, optionally
// filtered by state and labels and sliced into a page.
package list

import (
	"context"
	"fmt"
	"slices"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is list's input: which tasks to match, and which slice of the
// matches to return. The zero Request matches every task and returns all of
// them (Limit <= 0 means unlimited).
type Request struct {
	States []task.TaskState  `json:"state,omitempty"  jsonschema:"only tasks in one of these states"`
	Labels map[string]string `json:"label,omitempty"  jsonschema:"only tasks carrying every one of these labels"`
	Limit  int               `json:"limit,omitempty"  jsonschema:"max tasks to return; 0 for unlimited"`
	Offset int               `json:"offset,omitempty" jsonschema:"skip this many matching tasks before the page starts"`
}

func (r Request) validate() error {
	for _, state := range r.States {
		if _, err := task.ParseTaskState(state.String()); err != nil {
			return fmt.Errorf("state %q: %w", state, err)
		}
	}
	if r.Limit < 0 {
		return fmt.Errorf("limit must not be negative")
	}
	if r.Offset < 0 {
		return fmt.Errorf("offset must not be negative")
	}
	return nil
}

func (r Request) matches(t task.Task) bool {
	if len(r.States) > 0 && !slices.Contains(r.States, t.State()) {
		return false
	}
	for key, value := range r.Labels {
		if t.Definition.Labels[key] != value {
			return false
		}
	}
	return true
}

// List reads every task in st, keeps those matching req, and returns the page
// starting at req.Offset (all matches when req.Limit <= 0) together with the
// total number of matches before that slicing, so a caller can tell whether
// more pages remain.
func List(ctx context.Context, st *store.Store, req Request) ([]task.Task, int, error) {
	if err := req.validate(); err != nil {
		return nil, 0, fmt.Errorf("list: %w", err)
	}
	ids, err := st.List(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	matched := make([]task.Task, 0, len(ids))
	for _, id := range ids {
		t, err := st.Read(ctx, id)
		if err != nil {
			return nil, 0, fmt.Errorf("list tasks: %w", err)
		}
		if req.matches(t) {
			matched = append(matched, t)
		}
	}

	total := len(matched)
	start := min(max(req.Offset, 0), total)
	end := total
	if req.Limit > 0 && start+req.Limit < end {
		end = start + req.Limit
	}
	return matched[start:end], total, nil
}
