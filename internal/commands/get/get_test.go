package get

import (
	"errors"
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

func TestGet(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := st.Create(t.Context(), id, task.TaskDefinition{Description: "d"}, time.Now().UTC()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := Get(t.Context(), st, Request{ID: id})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != id {
		t.Fatalf("ID = %s, want %s", got.ID, id)
	}
}

func TestGetRejectsMissingTask(t *testing.T) {
	st := openTestStore(t)
	if _, err := Get(t.Context(), st, Request{ID: uuid.NewV7()}); !errors.Is(err, store.ErrTaskNotFound) {
		t.Fatalf("Get() error = %v, want ErrTaskNotFound", err)
	}
}
