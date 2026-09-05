package grep

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecute(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, []byte("package main\n// TODO: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := execute(t.Context(), input{Pattern: "TODO", Path: dir, Include: "*.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Matches) != 1 || got.Matches[0] != path+":2:// TODO: test" {
		t.Fatalf("matches = %v", got.Matches)
	}
}
