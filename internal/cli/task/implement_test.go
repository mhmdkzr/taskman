package task

import "testing"

func TestImplement(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	if _, err := runCmd(t, dir, Specify(), id, "--result", "spec", "--done-when", "criteria"); err != nil {
		t.Fatalf("specify: %v", err)
	}

	if _, err := runCmd(t, dir, Implement(), id); err != nil {
		t.Fatalf("implement: %v", err)
	}
	got := getJSON(t, dir, id)
	if got.Status.Implementation.State != "done" {
		t.Fatalf("implementation.state = %v, want done", got.Status.Implementation.State)
	}
}

func TestImplementRequiresSpecification(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	if _, err := runCmd(t, dir, Implement(), id); err == nil {
		t.Fatal("implement before specify: want error, got nil")
	}
}
