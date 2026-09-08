package task

import "testing"

func TestDelete(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)

	if _, err := runCmd(t, dir, Delete(), id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := runCmd(t, dir, Get(), id); err == nil {
		t.Fatal("get after delete: want error, got nil")
	}
}

func TestDeleteUnknownTask(t *testing.T) {
	dir := newTestRepo(t)
	if _, err := runCmd(t, dir, Delete(), "does-not-exist"); err == nil {
		t.Fatal("delete unknown task: want error, got nil")
	}
}
