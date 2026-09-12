package verified

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

func setUpToVerify(t *testing.T, st *store.Store) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	now := time.Now().UTC()
	if _, err := st.Create(id, task.TaskDefinition{Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := st.Append(id, task.SpecificationSubmitted{Specification: task.Specification{Plan: "p"}, At: now}); err != nil {
		t.Fatalf("Append(SpecificationSubmitted) error = %v", err)
	}
	if _, err := st.Append(id, task.ImplementationCompleted{
		Implementation: task.Implementation{Verification: task.Verification{Tests: task.TestConfiguration{Unit: true}}},
		At:             now,
	}); err != nil {
		t.Fatalf("Append(ImplementationCompleted) error = %v", err)
	}
	return id
}

func TestVerifiedDerivesPassed(t *testing.T) {
	st := openTestStore(t)
	id := setUpToVerify(t, st)

	got, err := Verified(st, Request{ID: id, Checks: task.Checks{Unit: task.CheckOK}})
	if err != nil {
		t.Fatalf("Verified() error = %v", err)
	}
	if got.State() != task.StateCommit {
		t.Fatalf("state = %s, want %s", got.State(), task.StateCommit)
	}
}

func TestVerifiedDerivesFailed(t *testing.T) {
	st := openTestStore(t)
	id := setUpToVerify(t, st)

	got, err := Verified(st, Request{ID: id, Checks: task.Checks{Unit: task.CheckError}})
	if err != nil {
		t.Fatalf("Verified() error = %v", err)
	}
	if got.State() != task.StateFixVerificationFailure {
		t.Fatalf("state = %s, want %s", got.State(), task.StateFixVerificationFailure)
	}
}

func TestVerifiedRejectsNoChecks(t *testing.T) {
	st := openTestStore(t)
	id := setUpToVerify(t, st)
	if _, err := Verified(st, Request{ID: id}); err == nil {
		t.Fatal("Verified() error = nil, want an error for no checks reported")
	}
}
