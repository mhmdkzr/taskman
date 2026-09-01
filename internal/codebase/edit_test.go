package codebase

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEditCreatesNewFile(t *testing.T) {
	repo, err := Open(newTestRepo(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	res, err := repo.Edit("new.txt", "", "hello\n", false)
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if !res.Created {
		t.Error("Created = false, want true")
	}

	root, err := repo.Root()
	if err != nil {
		t.Fatalf("Root: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, "new.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello\n" {
		t.Errorf("content = %q, want %q", got, "hello\n")
	}
}

func TestEditAppendsToExistingFile(t *testing.T) {
	repo, err := Open(newTestRepo(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if _, err := repo.Edit("README.md", "", "more\n", false); err != nil {
		t.Fatalf("Edit: %v", err)
	}

	root, _ := repo.Root()
	got, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello\nmore\n" {
		t.Errorf("content = %q, want %q", got, "hello\nmore\n")
	}
}

func TestEditReplacesUniqueMatch(t *testing.T) {
	repo, err := Open(newTestRepo(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if _, err := repo.Edit("README.md", "hello", "goodbye", false); err != nil {
		t.Fatalf("Edit: %v", err)
	}

	root, _ := repo.Root()
	got, _ := os.ReadFile(filepath.Join(root, "README.md"))
	if string(got) != "goodbye\n" {
		t.Errorf("content = %q, want %q", got, "goodbye\n")
	}
}

// TestEditAmbiguousMatchRejectedWithoutReplaceAll verifies old matching more
// than once is an error when replaceAll is false, so a caller can't silently
// touch the wrong occurrence.
func TestEditAmbiguousMatchRejectedWithoutReplaceAll(t *testing.T) {
	dir := newTestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("foo\nfoo\nfoo\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if _, err := repo.Edit("README.md", "foo", "bar", false); err == nil {
		t.Error("Edit with ambiguous old and replaceAll=false: want error")
	}
}

// TestEditReplaceAllReplacesEveryOccurrence verifies replaceAll=true replaces
// every match instead of erroring on ambiguity — the rename-a-symbol case.
func TestEditReplaceAllReplacesEveryOccurrence(t *testing.T) {
	dir := newTestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("foo\nfoo\nfoo\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if _, err := repo.Edit("README.md", "foo", "bar", true); err != nil {
		t.Fatalf("Edit: %v", err)
	}

	root, _ := repo.Root()
	got, _ := os.ReadFile(filepath.Join(root, "README.md"))
	if string(got) != "bar\nbar\nbar\n" {
		t.Errorf("content = %q, want %q", got, "bar\nbar\nbar\n")
	}
}

func TestEditNoMatchIsErrorEvenWithReplaceAll(t *testing.T) {
	repo, err := Open(newTestRepo(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if _, err := repo.Edit("README.md", "nope", "bar", true); err == nil {
		t.Error("Edit with no match: want error")
	}
}
