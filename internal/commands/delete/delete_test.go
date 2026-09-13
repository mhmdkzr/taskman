package delete

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

func TestDeleteRemovesTask(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := st.Create(t.Context(), id, task.TaskDefinition{Description: "d"}, time.Now().UTC()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := Delete(t.Context(), st, Request{ID: id})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if got.ID != id {
		t.Fatalf("Delete().ID = %s, want %s", got.ID, id)
	}
	if _, err := st.Read(t.Context(), id); !errors.Is(err, store.ErrTaskNotFound) {
		t.Fatalf("Read() after Delete() error = %v, want ErrTaskNotFound", err)
	}
}

func TestDeleteRejectsMissingID(t *testing.T) {
	st := openTestStore(t)
	if _, err := Delete(t.Context(), st, Request{}); err == nil {
		t.Fatal("Delete() error = nil, want an error for a missing id")
	}
}

func TestDeleteRejectsUnknownTask(t *testing.T) {
	st := openTestStore(t)
	if _, err := Delete(t.Context(), st, Request{ID: uuid.NewV7()}); !errors.Is(err, store.ErrTaskNotFound) {
		t.Fatalf("Delete() error = %v, want ErrTaskNotFound", err)
	}
}
