package codebase

import (
	"bytes"
	"fmt"
	"os/exec"
)

// GoDocArgs controls the flags passed to "go doc".
type GoDocArgs struct {
	// All shows all the documentation for the package.
	All bool

	// CaseSensitive respects case when matching symbols.
	CaseSensitive bool

	// Cmd treats a command (package main) like a regular package.
	Cmd bool

	// Short prints a one-line representation for each symbol.
	Short bool

	// Source shows the full source code for the symbol.
	Source bool

	// IncludeUnexported shows documentation for unexported symbols, methods, and fields.
	IncludeUnexported bool
}

func (c GoDocArgs) args() []string {
	var a []string
	if c.All {
		a = append(a, "-all")
	}
	if c.CaseSensitive {
		a = append(a, "-c")
	}
	if c.Cmd {
		a = append(a, "-cmd")
	}
	if c.Short {
		a = append(a, "-short")
	}
	if c.Source {
		a = append(a, "-src")
	}
	if c.IncludeUnexported {
		a = append(a, "-u")
	}
	return a
}

// GoDocQuery identifies the item to document: a package, a symbol within a package, or both.
type GoDocQuery struct {
	// Package is the import path of the package (e.g. "encoding/json").
	Package string

	// Symbol is the name of a symbol within the package (e.g. "Decoder.Decode").
	Symbol string
}

func (q GoDocQuery) args() []string {
	switch {
	case q.Package != "" && q.Symbol != "":
		return []string{q.Package, q.Symbol}
	case q.Package != "":
		return []string{q.Package}
	case q.Symbol != "":
		return []string{q.Symbol}
	default:
		return nil
	}
}

func (r Repository) GoDoc(cfg GoDocArgs, query GoDocQuery) (string, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return "", fmt.Errorf("worktree: %w", err)
	}

	allArgs := append(cfg.args(), query.args()...)
	cmd := exec.Command("go", append([]string{"doc"}, allArgs...)...)
	cmd.Dir = wt.Filesystem.Root()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go doc: %w: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
