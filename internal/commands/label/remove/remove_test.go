package remove

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
	st, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "tasks.db"))
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

func createTestTask(t *testing.T, st *store.Store, labels map[string]string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	if _, err := st.Create(
		t.Context(),
		id,
		task.TaskDefinition{Title: "t", Description: "d", Labels: labels},
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return id
}

func TestRemoveDeletesLabel(t *testing.T) {
	st := openTestStore(t)
	id := createTestTask(t, st, map[string]string{"priority": "high", "module": "wallet"})

	got, err := Remove(t.Context(), st, Request{ID: id, Keys: []string{"priority"}})
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, ok := got.Definition.Labels["priority"]; ok {
		t.Fatalf("labels = %v, want priority removed", got.Definition.Labels)
	}
	if got.Definition.Labels["module"] != "wallet" {
		t.Fatalf("labels = %v, want module=wallet kept", got.Definition.Labels)
	}
}

func TestRemoveLastLabelClearsMap(t *testing.T) {
	st := openTestStore(t)
	id := createTestTask(t, st, map[string]string{"priority": "high"})

	got, err := Remove(t.Context(), st, Request{ID: id, Keys: []string{"priority"}})
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if len(got.Definition.Labels) != 0 {
		t.Fatalf("labels = %v, want empty", got.Definition.Labels)
	}
}

func TestRemoveMissingKeyIsNotAnError(t *testing.T) {
	st := openTestStore(t)
	id := createTestTask(t, st, map[string]string{"priority": "high"})

	if _, err := Remove(t.Context(), st, Request{ID: id, Keys: []string{"nonexistent"}}); err != nil {
		t.Fatalf("Remove() error = %v, want nil for a key that isn't present", err)
	}
}

func TestRemoveRejectsEmptyKeys(t *testing.T) {
	st := openTestStore(t)
	id := createTestTask(t, st, nil)

	if _, err := Remove(t.Context(), st, Request{ID: id}); err == nil {
		t.Fatal("Remove() error = nil, want an error for no keys")
	}
}
