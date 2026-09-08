package task

import (
	"strings"
	"testing"
)

func TestNextDispatchesSpecifyAtCreation(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)

	out, err := runCmd(t, dir, Next(), id)
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if !strings.Contains(out, "specification") {
		t.Fatalf("next output = %q, want it to mention drafting a specification", out)
	}
}

func TestNextDoneAfterAbandon(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	if _, err := runCmd(t, dir, Abandon(), id, "--reason", "superseded"); err != nil {
		t.Fatalf("abandon: %v", err)
	}

	out, err := runCmd(t, dir, Next(), id)
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if !strings.Contains(out, "abandoned") || !strings.Contains(out, "superseded") {
		t.Fatalf("next output = %q", out)
	}
}
