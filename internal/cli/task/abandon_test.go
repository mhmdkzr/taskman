package task

import "testing"

func TestAbandon(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)

	if _, err := runCmd(t, dir, Abandon(), id, "--reason", "no longer needed"); err != nil {
		t.Fatalf("abandon: %v", err)
	}
	got := getJSON(t, dir, id)
	if got.State != "failed" || got.FailureReason != "no longer needed" {
		t.Fatalf("state = %v, reason = %q", got.State, got.FailureReason)
	}
}

func TestAbandonRequiresReason(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	if _, err := runCmd(t, dir, Abandon(), id); err == nil {
		t.Fatal("abandon without --reason: want error, got nil")
	}
}
