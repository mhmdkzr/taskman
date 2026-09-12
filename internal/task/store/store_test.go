package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return s
}

func TestCreateReadRoundTrip(t *testing.T) {
	s := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC().Truncate(time.Second)

	created, err := s.Create(t.Context(), id, task.TaskDefinition{Description: "do the work"}, now)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID != id {
		t.Fatalf("created.ID = %s, want %s", created.ID, id)
	}
	if created.State() != task.StateSpecify {
		t.Fatalf("created.State() = %s, want %s", created.State(), task.StateSpecify)
	}

	read, err := s.Read(t.Context(), id)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if read.ID != id || read.State() != task.StateSpecify {
		t.Fatalf("Read() = (%s, %s), want (%s, %s)", read.ID, read.State(), id, task.StateSpecify)
	}
}

func TestCreateRejectsDuplicateID(t *testing.T) {
	s := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC()

	if _, err := s.Create(t.Context(), id, task.TaskDefinition{Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := s.Create(t.Context(), id, task.TaskDefinition{Description: "d"}, now); !errors.Is(err, ErrTaskAlreadyExists) {
		t.Fatalf("Create() error = %v, want ErrTaskAlreadyExists", err)
	}
}

func TestReadRejectsMissingTask(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.Read(t.Context(), uuid.NewV7()); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("Read() error = %v, want ErrTaskNotFound", err)
	}
}

func TestAppendAccumulatesEventsAndMatchesDirectApply(t *testing.T) {
	s := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC().Truncate(time.Second)

	current, err := s.Create(t.Context(), id, task.TaskDefinition{Description: "do the work"}, now)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	events := []task.TaskEvent{
		task.SpecificationSubmitted{Specification: task.Specification{Plan: "plan"}, At: now},
		task.ImplementationCompleted{Implementation: task.Implementation{Git: task.Git{Worktree: "/wt", Branch: "b"}}, At: now},
		task.CommitRecorded{Commit: task.GitCommit{Hash: "abc", Message: "m", At: now}, At: now},
		task.MergeCompleted{Merge: task.GitMerge{Target: "main", Commit: "def", At: now}, At: now},
	}

	want := current
	for _, event := range events {
		want, err = task.Apply(want, event)
		if err != nil {
			t.Fatalf("task.Apply() error = %v", err)
		}
		current, err = s.Append(t.Context(), id, event)
		if err != nil {
			t.Fatalf("Append(%T) error = %v", event, err)
		}
		if current.State() != want.State() {
			t.Fatalf("Append() state = %s, want %s", current.State(), want.State())
		}
	}

	read, err := s.Read(t.Context(), id)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if read.State() != task.StateCompleted {
		t.Fatalf("Read().State() = %s, want %s", read.State(), task.StateCompleted)
	}
	if len(read.Implementation.Git.Commits) != 1 || read.Implementation.Git.Commits[0].Hash != "abc" {
		t.Fatalf("Read().Implementation.Git.Commits = %+v, want one commit 'abc'", read.Implementation.Git.Commits)
	}
}

func TestAppendRoundTripsEveryEventKind(t *testing.T) {
	s := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC().Truncate(time.Second)

	if _, err := s.Create(t.Context(), id, task.TaskDefinition{Description: "do the work"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	review := task.ReviewConfiguration{
		Agent: task.AgentReviewConfiguration{Required: true},
		Human: task.HumanReviewConfiguration{Required: true},
	}
	events := []task.TaskEvent{
		task.SpecificationSubmitted{Specification: task.Specification{Plan: "plan", Review: review}, At: now},
		task.SpecificationReviewAgentRejected{Findings: []task.Finding{{Location: "f", Detail: "d"}}, At: now},
		task.SpecificationSubmitted{Specification: task.Specification{Plan: "plan v2", Review: review}, At: now},
		task.SpecificationReviewAgentApproved{Comment: "ok", At: now},
		task.SpecificationReviewHumanRejected{Reason: "not yet", At: now},
		task.SpecificationSubmitted{Specification: task.Specification{Plan: "plan v3", Review: review}, At: now},
		task.SpecificationReviewAgentApproved{Comment: "ok", At: now},
		task.SpecificationReviewHumanApproved{Comment: "approved", At: now},
		task.ImplementationCompleted{
			Implementation: task.Implementation{
				Git:          task.Git{Worktree: "/wt", Branch: "b"},
				Verification: task.Verification{Tests: task.TestConfiguration{Unit: true}},
				Review:       review,
			},
			At: now,
		},
		task.VerificationFailed{Checks: task.Checks{Unit: task.CheckError}, At: now},
		task.VerificationPassed{Checks: task.Checks{Unit: task.CheckOK}, At: now},
		task.ImplementationReviewAgentRejected{Findings: []task.Finding{{Location: "g", Detail: "h"}}, At: now},
		task.VerificationPassed{Checks: task.Checks{Unit: task.CheckOK}, At: now},
		task.ImplementationReviewAgentApproved{Comment: "ok", At: now},
		task.CommitRecorded{Commit: task.GitCommit{Hash: "c1", Message: "m1", At: now}, At: now},
		task.ImplementationReviewHumanRejected{Reason: "needs work", At: now},
		task.VerificationPassed{Checks: task.Checks{Unit: task.CheckOK}, At: now},
		task.CommitRecorded{Commit: task.GitCommit{Hash: "c2", Message: "m2", At: now}, At: now},
		task.ImplementationReviewHumanApproved{Comment: "lgtm", At: now},
		task.MergeCompleted{Merge: task.GitMerge{Target: "main", Commit: "m", At: now}, At: now},
	}

	var current task.Task
	var err error
	for _, event := range events {
		current, err = s.Append(t.Context(), id, event)
		if err != nil {
			t.Fatalf("Append(%T) error = %v", event, err)
		}
	}
	if current.State() != task.StateCompleted {
		t.Fatalf("state = %s, want %s", current.State(), task.StateCompleted)
	}

	read, err := s.Read(t.Context(), id)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if read.State() != task.StateCompleted {
		t.Fatalf("Read().State() = %s, want %s", read.State(), task.StateCompleted)
	}
}

func TestEscalatedAndAbandonedRoundTrip(t *testing.T) {
	s := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC()

	if _, err := s.Create(t.Context(), id, task.TaskDefinition{Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := s.Append(t.Context(), id, task.Escalated{Stage: "definition", Reason: "stuck", At: now}); err != nil {
		t.Fatalf("Append(Escalated) error = %v", err)
	}
	current, err := s.Append(t.Context(), id, task.Abandoned{Reason: "moot", At: now})
	if err != nil {
		t.Fatalf("Append(Abandoned) error = %v", err)
	}
	if current.State() != task.StateAbandoned {
		t.Fatalf("state = %s, want %s", current.State(), task.StateAbandoned)
	}

	read, err := s.Read(t.Context(), id)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if read.Abandoned == nil || read.Abandoned.Reason != "moot" {
		t.Fatalf("Read().Abandoned = %+v, want reason 'moot'", read.Abandoned)
	}
}

func TestAppendRejectsInvalidEventWithoutPersistingIt(t *testing.T) {
	s := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC()

	if _, err := s.Create(t.Context(), id, task.TaskDefinition{Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// CommitRecorded is invalid from the initial specify state.
	if _, err := s.Append(t.Context(), id, task.CommitRecorded{Commit: task.GitCommit{Hash: "x", At: now}, At: now}); err == nil {
		t.Fatal("Append() error = nil, want an error for an invalid transition")
	}

	read, err := s.Read(t.Context(), id)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if read.State() != task.StateSpecify {
		t.Fatalf("Append() persisted a rejected event: state = %s, want %s", read.State(), task.StateSpecify)
	}
}

func TestListEnumeratesCreatedTasks(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC()
	id1, id2 := uuid.NewV7(), uuid.NewV7()

	if _, err := s.Create(t.Context(), id1, task.TaskDefinition{Description: "d1"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := s.Create(t.Context(), id2, task.TaskDefinition{Description: "d2"}, now.Add(time.Second)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	ids, err := s.List(t.Context())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(ids))
	}
	seen := map[uuid.UUID]bool{ids[0]: true, ids[1]: true}
	if !seen[id1] || !seen[id2] {
		t.Fatalf("List() = %v, want %v and %v", ids, id1, id2)
	}
}

func TestListOnEmptyStoreReturnsEmpty(t *testing.T) {
	s := openTestStore(t)
	ids, err := s.List(t.Context())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("List() = %v, want empty", ids)
	}
}

func TestReadRejectsUnknownEventKind(t *testing.T) {
	s := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC()

	if _, err := s.Create(t.Context(), id, task.TaskDefinition{Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	_, err := s.db.Exec(
		`INSERT INTO events (task_id, seq, kind, data) VALUES (?, 1, ?, ?)`,
		id.String(), "not_a_real_kind", `{}`,
	)
	if err != nil {
		t.Fatalf("insert corrupt event: %v", err)
	}

	if _, err := s.Read(t.Context(), id); !errors.Is(err, errCorruptLog) {
		t.Fatalf("Read() error = %v, want errCorruptLog", err)
	}
}
