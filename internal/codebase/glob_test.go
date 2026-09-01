package codebase

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestGlobMatchesDotfiles verifies Glob walks dotfiles/dot-directories like
// any other path (e.g. .golangci.yaml is a real config file a task's changes
// might need to see), skipping only the repository's own .git directory.
func TestGlobMatchesDotfiles(t *testing.T) {
	dir := newTestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, ".golangci.yaml"), []byte("linters: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".github", "workflows", "ci.yaml"), []byte("name: ci\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	matches, err := repo.Glob("**/*.yaml", ".")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	want := []string{".golangci.yaml", filepath.Join(".github", "workflows", "ci.yaml")}
	for _, w := range want {
		if !slices.Contains(matches, w) {
			t.Errorf("Glob matches = %v, want it to contain %q", matches, w)
		}
	}
}

// TestGlobSkipsGitDirectory verifies Glob never walks into .git, whose
// internal object/ref files would otherwise pollute any broad pattern match.
func TestGlobSkipsGitDirectory(t *testing.T) {
	dir := newTestRepo(t)
	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	matches, err := repo.Glob("**/*", ".")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	for _, m := range matches {
		if strings.HasPrefix(m, ".git"+string(filepath.Separator)) {
			t.Errorf("Glob matched inside .git: %q", m)
		}
	}
}
