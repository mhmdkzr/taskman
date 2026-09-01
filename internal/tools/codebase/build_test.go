package codebase

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/zendev-sh/goai"

	cb "github.com/mhmdkzr/taskman/internal/codebase"
)

// newGoTestRepo creates a scratch git repo containing a minimal, buildable
// Go module and returns the Repository wrapping it.
func newGoTestRepo(t *testing.T) cb.Repository {
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
		"main_test.go": `package main

import "testing"

func TestOK(t *testing.T) {}
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

	r, err := cb.Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return r
}

func execTool(t *testing.T, tool goai.Tool) string {
	t.Helper()
	out, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("%s: %v", tool.Name, err)
	}
	return out
}

func TestGoBuildToolSuccess(t *testing.T) {
	repo := newGoTestRepo(t)
	if out := execTool(t, GoBuildTool(repo)); out != "build ok" {
		t.Errorf("go_build = %q, want %q", out, "build ok")
	}
}

func TestGoBuildToolFailure(t *testing.T) {
	repo := newGoTestRepo(t)
	root, err := repo.Root()
	if err != nil {
		t.Fatalf("Root: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "broken.go"),
		[]byte("package main\n\nfunc broken( {\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	out := execTool(t, GoBuildTool(repo))
	if !strings.Contains(out, "build failed") {
		t.Errorf("go_build = %q, want it to report a failure", out)
	}
}

func TestGoTestToolReportsPass(t *testing.T) {
	repo := newGoTestRepo(t)
	out := execTool(t, GoTestTool(repo))
	if !strings.Contains(out, "1 package(s) passed, 0 failed") {
		t.Errorf("go_test = %q, want a 1-passed-0-failed summary", out)
	}
}

func TestGoTestToolReportsFailure(t *testing.T) {
	repo := newGoTestRepo(t)
	root, err := repo.Root()
	if err != nil {
		t.Fatalf("Root: %v", err)
	}
	failing := `package main

import "testing"

func TestAlwaysFails(t *testing.T) {
	t.Fatal("boom")
}
`
	if err := os.WriteFile(filepath.Join(root, "fail_test.go"), []byte(failing), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	out := execTool(t, GoTestTool(repo))
	if !strings.Contains(out, "boom") || !strings.Contains(out, "0 package(s) passed, 1 failed") {
		t.Errorf("go_test = %q, want it to surface the failure and a 0-passed-1-failed summary", out)
	}
}
