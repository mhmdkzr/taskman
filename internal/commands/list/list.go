// Package list owns the "list" command: it enumerates every task.
package list

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// List reads every task in st.
func List(ctx context.Context, st *store.Store) ([]task.Task, error) {
	ids, err := st.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	tasks := make([]task.Task, 0, len(ids))
	for _, id := range ids {
		t, err := st.Read(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("list tasks: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
