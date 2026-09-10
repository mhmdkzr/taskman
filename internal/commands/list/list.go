// Package list owns the "list" command: its domain logic and CLI wiring.
package list

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// defaultLimit is the page size a frontend applies when the caller doesn't
// give one, so listing against a large tasks-dir doesn't dump everything at
// once by default.
const defaultLimit = 50

// Request narrows and pages List's results. Shared verbatim by the CLI
// (cmd.go builds it from flags) and MCP (mcp.go uses it as the tool's input
// type directly) frontends; the jsonschema tags describe it to MCP clients.
// A zero Request matches every task (Limit <= 0 means unlimited).
type Request struct {
	State  []task.State      `json:"state,omitempty"  jsonschema:"filter by workflow state"`
	Labels map[string]string `json:"labels,omitempty" jsonschema:"filter by label key/value pairs"`
	Limit  int               `json:"limit,omitempty"  jsonschema:"max tasks to return; 0 or negative means unlimited"`
	Offset int               `json:"offset,omitempty" jsonschema:"skip this many matching tasks before the page starts"`
}

// Result is List's output shape: the page of tasks plus enough to tell
// whether more pages remain.
type Result struct {
	Tasks  []task.Task `json:"tasks"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

func (r Request) matches(t task.Task) bool {
	if len(r.State) > 0 {
		found := slices.Contains(r.State, t.State)
		if !found {
			return false
		}
	}
	for k, v := range r.Labels {
		if t.Labels[k] != v {
			return false
		}
	}
	return true
}

// List returns the page of tasks matching req under tasksDir - ordered by
// id, sliced to [Offset, Offset+Limit) (the whole match if Limit <= 0) - and
// the total number of tasks matching req before that slicing, so a caller
// can tell whether more pages remain.
func List(tasksDir string, req Request) (Result, error) {
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{Limit: req.Limit, Offset: req.Offset}, nil
		}
		return Result{}, fmt.Errorf("read tasks dir: %w", err)
	}

	var ids []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(entry.Name(), ".yaml"))
	}
	sort.Strings(ids)

	tasks := make([]task.Task, 0, len(ids))
	for _, id := range ids {
		t, err := store.Read(tasksDir, id)
		if err != nil {
			return Result{}, fmt.Errorf("read task: %w", err)
		}
		if req.matches(t) {
			tasks = append(tasks, t)
		}
	}

	total := len(tasks)
	start := min(max(req.Offset, 0), total)
	end := total
	if req.Limit > 0 && start+req.Limit < end {
		end = start + req.Limit
	}
	return Result{Tasks: tasks[start:end], Total: total, Limit: req.Limit, Offset: req.Offset}, nil
}
