package codebase

import (
	"fmt"
	"os/exec"
)

// GoFmt reformats every Go file in the module's own packages (see
// sourceFiles) — vendor/ is never touched, matching what go build/vet and
// every lint tool in this package already limit themselves to.
func (r Repository) GoFmt() error {
	wt, err := r.r.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}
	files, err := r.sourceFiles()
	if err != nil {
		return fmt.Errorf("source files: %w", err)
	}
	if len(files) == 0 {
		return nil
	}

	cmd := exec.Command("gofmt", append([]string{"-w"}, files...)...)
	cmd.Dir = wt.Filesystem.Root()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gofmt: %w: %s", err, out)
	}

	return nil
}
