package codebase

import (
	"fmt"
	"os/exec"
)

func (r Repository) GoFmt() error {
	wt, err := r.r.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	cmd := exec.Command("gofmt", "-w", ".")
	cmd.Dir = wt.Filesystem.Root()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gofmt: %w: %s", err, out)
	}

	return nil
}
