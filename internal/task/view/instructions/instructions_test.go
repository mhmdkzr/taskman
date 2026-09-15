package instructions

import (
	"path/filepath"
	"strings"
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

func createTask(t *testing.T, st *store.Store) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	definition := task.TaskDefinition{Title: "T", Description: "d"}
	if _, err := st.Create(t.Context(), id, definition, time.Now().UTC()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return id
}

func TestProjectAtSpecifyBeforeSpecOmitsFeedback(t *testing.T) {
	st := openTestStore(t)
	id := createTask(t, st)

	got := readAndProject(t, st, id)
	if got.State != task.StateSpecify || got.Action != task.InstructionDispatch {
		t.Fatalf("State/Action = %s/%s", got.State, got.Action)
	}
	if strings.Contains(got.Message, "Address this feedback") {
		t.Fatalf("Message = %q, want no feedback block before any review", got.Message)
	}
}

func TestProjectAtSpecifyAfterRejectionAddressesFindings(t *testing.T) {
	st := openTestStore(t)
	id := createTask(t, st)
	now := time.Now().UTC()
	review := task.ReviewConfiguration{
		Agent: task.AgentReviewConfiguration{Required: true},
		Human: task.HumanReviewConfiguration{Required: true},
	}
	if _, err := st.Append(t.Context(), id, task.SpecificationSubmitted{
		Specification: task.Specification{Plan: "p", Review: review}, At: now,
	}); err != nil {
		t.Fatalf("Append(SpecificationSubmitted) error = %v", err)
	}
	if _, err := st.Append(t.Context(), id, task.SpecificationReviewAgentRejected{
		Findings: []task.Finding{{Location: "plan", Detail: "no fallback"}}, At: now,
	}); err != nil {
		t.Fatalf("Append(SpecificationReviewAgentRejected) error = %v", err)
	}

	got := readAndProject(t, st, id)
	if got.State != task.StateSpecify || got.Action != task.InstructionDispatch {
		t.Fatalf("State/Action = %s/%s", got.State, got.Action)
	}
	for _, want := range []string{"Address this feedback", "Automated specification review rejected this plan:", "plan: no fallback"} {
		if !strings.Contains(got.Message, want) {
			t.Fatalf("Message = %q, want it to contain %q", got.Message, want)
		}
	}
}

func readAndProject(t *testing.T, st *store.Store, id uuid.UUID) Instructions {
	t.Helper()
	task, err := st.Read(t.Context(), id)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	got, err := Project(task)
	if err != nil {
		t.Fatalf("Project() error = %v", err)
	}
	return got
}

func TestProjectAtSpecifySuggestsSpecified(t *testing.T) {
	st := openTestStore(t)
	id := createTask(t, st)

	got := readAndProject(t, st, id)
	if got.State != task.StateSpecify || got.Action != task.InstructionDispatch {
		t.Fatalf("State/Action = %s/%s, want %s/%s", got.State, got.Action, task.StateSpecify, task.InstructionDispatch)
	}
	if len(got.Commands) != 1 || !strings.HasPrefix(got.Commands[0], "specified --id ") {
		t.Fatalf("Commands = %v, want one specified command", got.Commands)
	}
}

func TestProjectAtSpecificationReviewListsBothGates(t *testing.T) {
	st := openTestStore(t)
	id := createTask(t, st)
	now := time.Now().UTC()
	review := task.ReviewConfiguration{
		Agent: task.AgentReviewConfiguration{Required: true},
		Human: task.HumanReviewConfiguration{Required: true},
	}
	if _, err := st.Append(t.Context(), id, task.SpecificationSubmitted{
		Specification: task.Specification{Plan: "p", Review: review}, At: now,
	}); err != nil {
		t.Fatalf("Append(SpecificationSubmitted) error = %v", err)
	}

	got := readAndProject(t, st, id)
	if got.State != task.StateSpecificationReview || got.Action != task.InstructionWait {
		t.Fatalf("State/Action = %s/%s", got.State, got.Action)
	}
	if len(got.Commands) != 4 {
		t.Fatalf("Commands = %v, want 4 (agent+human, approve+reject)", got.Commands)
	}
}

func TestProjectAtVerifyCollapsesPassAndFail(t *testing.T) {
	st := openTestStore(t)
	id := createTask(t, st)
	now := time.Now().UTC()
	if _, err := st.Append(t.Context(), id, task.SpecificationSubmitted{
		Specification: task.Specification{Plan: "p"}, At: now,
	}); err != nil {
		t.Fatalf("Append(SpecificationSubmitted) error = %v", err)
	}
	if _, err := st.Append(t.Context(), id, task.ImplementationCompleted{
		Implementation: task.Implementation{
			Git:          task.Git{Worktree: "/wt", Branch: "feat/x"},
			Verification: task.Verification{Tests: task.TestConfiguration{Unit: true}},
		},
		At: now,
	}); err != nil {
		t.Fatalf("Append(ImplementationCompleted) error = %v", err)
	}

	got := readAndProject(t, st, id)
	if got.State != task.StateVerify {
		t.Fatalf("State = %s, want %s", got.State, task.StateVerify)
	}
	if len(got.Commands) != 1 || !strings.HasPrefix(got.Commands[0], "verified --id ") {
		t.Fatalf("Commands = %v, want one collapsed verified command", got.Commands)
	}
	if !strings.Contains(got.Message, "/wt") || !strings.Contains(got.Message, "feat/x") {
		t.Fatalf("Message = %q, want it to mention worktree/branch", got.Message)
	}
}

func TestProjectAtFixVerificationFailureExplainsWhy(t *testing.T) {
	st := openTestStore(t)
	id := createTask(t, st)
	now := time.Now().UTC()
	if _, err := st.Append(t.Context(), id, task.SpecificationSubmitted{
		Specification: task.Specification{Plan: "p"}, At: now,
	}); err != nil {
		t.Fatalf("Append(SpecificationSubmitted) error = %v", err)
	}
	if _, err := st.Append(t.Context(), id, task.ImplementationCompleted{
		Implementation: task.Implementation{
			Git:          task.Git{Worktree: "/wt", Branch: "feat/x"},
			Verification: task.Verification{Tests: task.TestConfiguration{Unit: true}},
		},
		At: now,
	}); err != nil {
		t.Fatalf("Append(ImplementationCompleted) error = %v", err)
	}
	if _, err := st.Append(t.Context(), id, task.VerificationFailed{
		Checks: task.Checks{Unit: task.CheckError}, Output: "boom", At: now,
	}); err != nil {
		t.Fatalf("Append(VerificationFailed) error = %v", err)
	}

	got := readAndProject(t, st, id)
	if got.State != task.StateFixVerificationFailure {
		t.Fatalf("State = %s, want %s", got.State, task.StateFixVerificationFailure)
	}
	if !strings.Contains(got.Message, "unit") || !strings.Contains(got.Message, "boom") {
		t.Fatalf("Message = %q, want it to mention the failed check and output", got.Message)
	}
}

func TestProjectAtCompletedHasNoCommands(t *testing.T) {
	st := openTestStore(t)
	id := createTask(t, st)
	now := time.Now().UTC()
	if _, err := st.Append(t.Context(), id, task.SpecificationSubmitted{
		Specification: task.Specification{Plan: "p"}, At: now,
	}); err != nil {
		t.Fatalf("Append(SpecificationSubmitted) error = %v", err)
	}
	if _, err := st.Append(t.Context(), id, task.ImplementationCompleted{
		Implementation: task.Implementation{Git: task.Git{Worktree: "/wt", Branch: "feat/x"}},
		At:             now,
	}); err != nil {
		t.Fatalf("Append(ImplementationCompleted) error = %v", err)
	}
	if _, err := st.Append(t.Context(), id, task.CommitRecorded{
		Commit: task.GitCommit{Hash: "abc", Message: "m", At: now}, At: now,
	}); err != nil {
		t.Fatalf("Append(CommitRecorded) error = %v", err)
	}
	if _, err := st.Append(t.Context(), id, task.MergeCompleted{
		Merge: task.GitMerge{Target: "main", Commit: "def", At: now}, At: now,
	}); err != nil {
		t.Fatalf("Append(MergeCompleted) error = %v", err)
	}

	got := readAndProject(t, st, id)
	if got.State != task.StateCompleted || got.Action != task.InstructionDone {
		t.Fatalf("State/Action = %s/%s", got.State, got.Action)
	}
	if len(got.Commands) != 0 {
		t.Fatalf("Commands = %v, want none", got.Commands)
	}
}
