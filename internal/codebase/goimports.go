package codebase

import (
	"fmt"
	"os/exec"
)

func (r Repository) GoImports() error {
	wt, err := r.r.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	cmd := exec.Command("goimports", "-w", ".")
	cmd.Dir = wt.Filesystem.Root()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("goimports: %w: %s", err, out)
	}

	return nil
}
