package codebase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"go/types"
)

type LoC struct {
	total    uint64
	tests    uint64
	comments uint64
}

type sccFile struct {
	Location string `json:"Location"`
	Lines    uint64 `json:"Lines"`
	Code     uint64 `json:"Code"`
	Comment  uint64 `json:"Comment"`
	Blank    uint64 `json:"Blank"`
}

type sccLanguage struct {
	Name    string    `json:"Name"`
	Lines   uint64    `json:"Lines"`
	Code    uint64    `json:"Code"`
	Comment uint64    `json:"Comment"`
	Blank   uint64    `json:"Blank"`
	Files   []sccFile `json:"Files"`
}

func (r Repository) GetLoC() (LoC, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return LoC{}, fmt.Errorf("worktree: %w", err)
	}
	return runSCC(wt.Filesystem.Root())
}

func (r Repository) GetPkgLoC(pkg *types.Package) (LoC, error) {
	pkgs, err := r.GoList(pkg.Path())
	if err != nil {
		return LoC{}, fmt.Errorf("go list: %w", err)
	}
	if len(pkgs) == 0 {
		return LoC{}, fmt.Errorf("package %q not found", pkg.Path())
	}
	return runSCC(pkgs[0].Dir)
}

func runSCC(dir string) (LoC, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("scc", "--include-ext", "go", "--format", "json", dir)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return LoC{}, fmt.Errorf("scc: %w: %s", err, stderr.String())
	}

	var langs []sccLanguage
	if err := json.Unmarshal(stdout.Bytes(), &langs); err != nil {
		return LoC{}, fmt.Errorf("parse scc output: %w", err)
	}

	var loc LoC
	for _, lang := range langs {
		loc.comments += lang.Comment
		loc.total += lang.Code
		for _, f := range lang.Files {
			if strings.HasSuffix(f.Location, "_test.go") {
				loc.tests += f.Code
			}
		}
	}
	return loc, nil
}
