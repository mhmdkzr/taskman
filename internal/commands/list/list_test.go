package list

import (
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return st
}

func TestList(t *testing.T) {
	st := openTestStore(t)
	now := time.Now().UTC()
	if _, err := st.Create(t.Context(), uuid.NewV7(), task.TaskDefinition{Description: "d1"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := st.Create(
		t.Context(),
		uuid.NewV7(),
		task.TaskDefinition{Description: "d2"},
		now.Add(time.Second),
	); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	tasks, err := List(t.Context(), st)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(tasks))
	}
}

func TestListOnEmptyStoreReturnsEmpty(t *testing.T) {
	st := openTestStore(t)
	tasks, err := List(t.Context(), st)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("List() = %v, want empty", tasks)
	}
}
