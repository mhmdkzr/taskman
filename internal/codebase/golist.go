package codebase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
)

type Package struct {
	Dir         string        `json:"Dir"`
	ImportPath  string        `json:"ImportPath"`
	Name        string        `json:"Name"`
	GoFiles     []string      `json:"GoFiles"`
	Imports     []string      `json:"Imports"`
	Deps        []string      `json:"Deps"`
	TestGoFiles []string      `json:"TestGoFiles"`
	TestImports []string      `json:"TestImports"`
	Error       *PackageError `json:"Error,omitempty"`
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
