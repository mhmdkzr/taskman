package task

import "testing"

func TestSpecify(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)

	if _, err := runCmd(t, dir, Specify(), id, "--result", "the spec", "--done-when", "criteria met"); err != nil {
		t.Fatalf("specify: %v", err)
	}
	got := getJSON(t, dir, id)
	if got.Specification != "the spec" || got.DoneWhen != "criteria met" {
		t.Fatalf("specification = %q, done_when = %q", got.Specification, got.DoneWhen)
	}
	if got.State != "started" {
		t.Fatalf("state = %v, want started", got.State)
	}
}

func TestSpecifyRequiresFlags(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	if _, err := runCmd(t, dir, Specify(), id, "--result", "only result"); err == nil {
		t.Fatal("specify without --done-when: want error, got nil")
	}
}

func TestSpecifyAlreadyDone(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	if _, err := runCmd(t, dir, Specify(), id, "--result", "r", "--done-when", "d"); err != nil {
		t.Fatalf("specify: %v", err)
	}
	if _, err := runCmd(t, dir, Specify(), id, "--result", "r2", "--done-when", "d2"); err == nil {
		t.Fatal("re-specify: want error, got nil")
	}
}
