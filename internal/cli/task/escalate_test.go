package task

import "testing"

func TestEscalate(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)

	if _, err := runCmd(
		t,
		dir,
		Escalate(),
		id,
		"--stage",
		"implementation",
		"--reason",
		"ambiguous requirements",
	); err != nil {
		t.Fatalf("escalate: %v", err)
	}
	got := getJSON(t, dir, id)
	if got.State != "blocked" || got.Blocked == nil || got.Blocked.Stage != "implementation" {
		t.Fatalf("state = %v, blocked = %+v", got.State, got.Blocked)
	}
}

func TestEscalateRequiresFlags(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	if _, err := runCmd(t, dir, Escalate(), id, "--stage", "implementation"); err == nil {
		t.Fatal("escalate without --reason: want error, got nil")
	}
}

func TestEscalateRefusesTerminalTask(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	if _, err := runCmd(t, dir, Abandon(), id, "--reason", "done with it"); err != nil {
		t.Fatalf("abandon: %v", err)
	}
	if _, err := runCmd(t, dir, Escalate(), id, "--stage", "implementation", "--reason", "x"); err == nil {
		t.Fatal("escalate on a failed task: want error, got nil")
	}
}
