package codebase

import (
	"fmt"
	"os/exec"
)

func (r Repository) GoGet(pkgs ...string) error {
	wt, err := r.r.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	args := append([]string{"get"}, pkgs...)
	cmd := exec.Command("go", args...)
	cmd.Dir = wt.Filesystem.Root()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go get: %w: %s", err, out)
	}

	return nil
}
