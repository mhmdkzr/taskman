// Package list owns the "list" command: its domain logic and CLI wiring.
package list

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/mhmdkzr/loop/internal/task"
)

// Filter narrows List's results. A zero Filter matches every task.
type Filter struct {
	State  []task.State
	Labels map[string]string
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

// List returns every task matching filter under tasksDir, ordered by id.
func List(tasksDir string, filter Filter) ([]task.Task, error) {
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tasks dir: %w", err)
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
			return nil, fmt.Errorf("read task: %w", err)
		}
		if filter.matches(t) {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}
