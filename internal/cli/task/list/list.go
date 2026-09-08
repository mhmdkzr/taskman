// Package list owns the "list" command: its domain logic and CLI wiring.
package list

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Filter narrows List's results. A zero Filter matches every task and
// returns all of them (Limit <= 0 means unlimited).
type Filter struct {
	State  []task.State
	Labels map[string]string
	Limit  int
	Offset int
}

func (f Filter) matches(t task.Task) bool {
	if len(f.State) > 0 {
		found := slices.Contains(f.State, t.State)
		if !found {
			return false
		}
	}
	for k, v := range f.Labels {
		if t.Labels[k] != v {
			return false
		}
	}
	return true
}

// List returns the page of tasks matching filter under tasksDir - ordered
// by id, sliced to [Offset, Offset+Limit) (the whole match if Limit <= 0) -
// and the total number of tasks matching filter before that slicing, so a
// caller can tell whether more pages remain.
func List(tasksDir string, filter Filter) ([]task.Task, int, error) {
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("read tasks dir: %w", err)
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
		t, err := task.ReadTask(tasksDir, id)
		if err != nil {
			return nil, 0, fmt.Errorf("read task: %w", err)
		}
		if filter.matches(t) {
			tasks = append(tasks, t)
		}
	}

	total := len(tasks)
	start := min(max(filter.Offset, 0), total)
	end := total
	if filter.Limit > 0 && start+filter.Limit < end {
		end = start + filter.Limit
	}
	return tasks[start:end], total, nil
}
