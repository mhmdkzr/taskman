package codebase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
)

type Package struct {
	Dir          string        `json:"Dir"`
	ImportPath   string        `json:"ImportPath"`
	Name         string        `json:"Name"`
	GoFiles      []string      `json:"GoFiles"`
	Imports      []string      `json:"Imports"`
	Deps         []string      `json:"Deps"`
	TestGoFiles  []string      `json:"TestGoFiles"`
	XTestGoFiles []string      `json:"XTestGoFiles"`
	TestImports  []string      `json:"TestImports"`
	Error        *PackageError `json:"Error,omitempty"`
}

type PackageError struct {
	ImportStack []string `json:"ImportStack"`
	Pos         string   `json:"Pos"`
	Err         string   `json:"Err"`
}

func (p Package) HasTests() bool {
	return len(p.TestGoFiles) > 0
}

func (p Package) Failed() bool {
	return p.Error != nil
}

func (r Repository) GoList(patterns ...string) ([]Package, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	args := append([]string{"list", "-json"}, patterns...)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", args...)
	cmd.Dir = wt.Filesystem.Root()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("go list: %w: %s", err, stderr.String())
	}

	pkgs, err := ParseGoListOutput(&stdout)
	if err != nil {
		return nil, fmt.Errorf("parse go list output: %w", err)
	}

	return pkgs, nil
}

// sourceFiles returns the worktree-relative path of every .go file in every
// package Go itself considers part of the module — i.e. exactly what
// `go build ./...`, `go vet ./...`, staticcheck, and golangci-lint already
// operate on, which in a vendored module excludes vendor/ (go list resolves
// it as dependencies, not packages `./...` walks).
//
// gofmt and goimports have no such awareness, and — this is the part a
// directory-level filter still gets wrong — given any directory argument
// they recurse into every subdirectory beneath it, not just the files
// directly inside it. A package's own Dir is frequently the module root
// itself (e.g. this module's own main package), so passing package
// directories through would still walk straight into vendor/ under it.
// Passing the explicit file list instead is the only way to guarantee
// gofmt/goimports never touch anything outside what Go itself considers
// source.
func (r Repository) sourceFiles() ([]string, error) {
	root, err := r.root()
	if err != nil {
		return nil, err
	}
	pkgs, err := r.GoList()
	if err != nil {
		return nil, err
	}
	var files []string
	for _, p := range pkgs {
		if p.Dir == "" {
			continue
		}
		relDir, err := filepath.Rel(root, p.Dir)
		if err != nil {
			return nil, fmt.Errorf("relativize %s: %w", p.Dir, err)
		}
		for _, names := range [][]string{p.GoFiles, p.TestGoFiles, p.XTestGoFiles} {
			for _, name := range names {
				files = append(files, filepath.Join(relDir, name))
			}
		}
	}
	return files, nil
}

func ParseGoListOutput(r io.Reader) ([]Package, error) {
	var pkgs []Package
	dec := json.NewDecoder(r)
	for dec.More() {
		var p Package
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("parsing package: %w", err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}
