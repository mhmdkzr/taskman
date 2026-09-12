package json

import (
	"encoding/json"
	"testing"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
)

func mustApply(t *testing.T, current task.Task, event task.TaskEvent) task.Task {
	t.Helper()
	next, err := task.Apply(current, event)
	if err != nil {
		t.Fatalf("task.Apply(%T) error = %v", event, err)
	}
	return next
}

func TestFromTaskJustCreated(t *testing.T) {
	now := time.Now().UTC()
	tsk, err := task.NewTask(uuid.NewV7(), task.TaskDefinition{Title: "t", Description: "d"}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}

	doc := FromTask(tsk)
	if doc.State != task.StateSpecify {
		t.Fatalf("State = %s, want %s", doc.State, task.StateSpecify)
	}
	if doc.Instruction.Action != task.InstructionDispatch {
		t.Fatalf("Instruction.Action = %s, want %s", doc.Instruction.Action, task.InstructionDispatch)
	}
	if doc.Instruction.State != task.StateSpecify {
		t.Fatalf("Instruction.State = %s, want %s", doc.Instruction.State, task.StateSpecify)
	}
	if doc.Task.ID != tsk.ID {
		t.Fatalf("Task.ID = %s, want %s", doc.Task.ID, tsk.ID)
	}
}

func TestFromTaskCompleted(t *testing.T) {
	now := time.Now().UTC()
	tsk, err := task.NewTask(uuid.NewV7(), task.TaskDefinition{Description: "d"}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	tsk = mustApply(t, tsk, task.SpecificationSubmitted{Specification: task.Specification{Plan: "p"}, At: now})
	tsk = mustApply(
		t,
		tsk,
		task.ImplementationCompleted{
			Implementation: task.Implementation{Git: task.Git{Worktree: "/wt", Branch: "b"}},
			At:             now,
		},
	)
	tsk = mustApply(t, tsk, task.CommitRecorded{Commit: task.GitCommit{Hash: "c", At: now}, At: now})
	tsk = mustApply(t, tsk, task.MergeCompleted{Merge: task.GitMerge{Target: "main", Commit: "c", At: now}, At: now})

	doc := FromTask(tsk)
	if doc.State != task.StateCompleted {
		t.Fatalf("State = %s, want %s", doc.State, task.StateCompleted)
	}
	if doc.Instruction.Action != task.InstructionDone {
		t.Fatalf("Instruction.Action = %s, want %s", doc.Instruction.Action, task.InstructionDone)
	}
}

func TestFromTaskBlocked(t *testing.T) {
	now := time.Now().UTC()
	tsk, err := task.NewTask(uuid.NewV7(), task.TaskDefinition{Description: "d"}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	tsk = mustApply(t, tsk, task.Escalated{Stage: "definition", Reason: "unclear", At: now})

	doc := FromTask(tsk)
	if doc.State != task.StateBlocked {
		t.Fatalf("State = %s, want %s", doc.State, task.StateBlocked)
	}
	if doc.Instruction.Action != task.InstructionWait {
		t.Fatalf("Instruction.Action = %s, want %s", doc.Instruction.Action, task.InstructionWait)
	}
}

func TestDocumentJSONShape(t *testing.T) {
	now := time.Now().UTC()
	tsk, err := task.NewTask(uuid.NewV7(), task.TaskDefinition{Title: "t", Description: "d"}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}

	data, err := json.Marshal(FromTask(tsk))
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got["state"] != "specify" {
		t.Fatalf("state = %v, want %q", got["state"], "specify")
	}
	instruction, ok := got["instruction"].(map[string]any)
	if !ok {
		t.Fatalf("instruction = %T, want an object", got["instruction"])
	}
	if instruction["state"] != "specify" || instruction["action"] != "dispatch" {
		t.Fatalf("instruction = %v, want {state: specify, action: dispatch}", instruction)
	}
	inner, ok := got["task"].(map[string]any)
	if !ok {
		t.Fatalf("task = %T, want an object", got["task"])
	}
	if _, ok := inner["state-history"]; !ok {
		t.Fatalf("task document should still carry the state-history, got %v", inner)
	}
}
