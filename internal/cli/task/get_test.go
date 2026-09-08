package task

import (
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)

	out, err := runCmd(t, dir, Get(), id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out, id) {
		t.Errorf("get output = %q, want it to mention %q", out, id)
	}
}

func TestGetMissingID(t *testing.T) {
	dir := newTestRepo(t)
	if _, err := runCmd(t, dir, Get()); err == nil {
		t.Fatal("get without an id: want error, got nil")
	}
}

func TestGetUnknownTask(t *testing.T) {
	dir := newTestRepo(t)
	if _, err := runCmd(t, dir, Get(), "does-not-exist"); err == nil {
		t.Fatal("get unknown task: want error, got nil")
	}
}
