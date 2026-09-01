package codebase

import (
	"fmt"
	"os/exec"
)

func (r Repository) GoModTidy() error {
	wt, err := r.r.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = wt.Filesystem.Root()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod tidy: %w: %s", err, out)
	}

	return nil
}
