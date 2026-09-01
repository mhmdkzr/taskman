package codebase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// newGoRepo creates a scratch git repo containing a minimal, buildable Go
// module and returns the Repository wrapping it.
func newGoRepo(t *testing.T) Repository {
	t.Helper()
	dir := t.TempDir()

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}

	files := map[string]string{
		"go.mod": "module example.com/scratch\n\ngo 1.24\n",
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("hi")
}
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}
	if err := wt.AddGlob("."); err != nil {
		t.Fatalf("AddGlob: %v", err)
	}
	sig := &object.Signature{Name: "Test", Email: "test@example.com"}
	if _, err := wt.Commit("initial commit", &git.CommitOptions{Author: sig}); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return r
}

func TestFormatAppliesGofmtAndGoimports(t *testing.T) {
	repo := newGoRepo(t)
	root, err := repo.Root()
	if err != nil {
		t.Fatalf("Root: %v", err)
	}
	messy := "package main\n\nimport (\n\"fmt\"\n)\n\nfunc Messy(){fmt.Println(\"x\")}\n"
	if err := os.WriteFile(filepath.Join(root, "messy.go"), []byte(messy), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := repo.Format(); err != nil {
		t.Fatalf("Format: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "messy.go"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(got), `func Messy() { fmt.Println("x") }`) {
		t.Errorf("messy.go not reformatted, got:\n%s", got)
	}
}
