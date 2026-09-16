package unblocked

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

func TestUnblocked(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := st.Create(
		t.Context(),
		id,
		task.TaskDefinition{Title: "t", Description: "d"},
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := st.Append(t.Context(), id, task.Escalated{Stage: "definition", Reason: "stuck"}); err != nil {
		t.Fatalf("Append(Escalated) error = %v", err)
	}

	got, err := Unblocked(t.Context(), st, Request{ID: id, Reason: "input received"})
	if err != nil {
		t.Fatalf("Unblocked() error = %v", err)
	}
	if got.State() != task.StateSpecify {
		t.Fatalf("state = %s, want %s", got.State(), task.StateSpecify)
	}
	if got.Blocked != nil {
		t.Fatalf("Blocked = %+v, want nil after unblocking", got.Blocked)
	}
}

func TestUnblockedRejectsMissingReason(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := Unblocked(t.Context(), st, Request{ID: id}); err == nil {
		t.Fatal("Unblocked() error = nil, want an error for a missing reason")
	}
}
