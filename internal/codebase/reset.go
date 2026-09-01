package codebase

import (
	"errors"
	"fmt"
	"io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func (r Repository) Reset() error {
	head, err := r.r.Head()
	if err != nil {
		return fmt.Errorf("head: %w", err)
	}

	wt, err := r.r.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	err = wt.Reset(&git.ResetOptions{
		Mode:   git.HardReset,
		Commit: head.Hash(),
	})
	if err != nil {
		return fmt.Errorf("reset: %w", err)
	}

	return nil
}

func (r Repository) ResetFile(path string) error {
	head, err := r.r.Head()
	if err != nil {
		return fmt.Errorf("head: %w", err)
	}

	commit, err := r.r.CommitObject(head.Hash())
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("tree: %w", err)
	}

	file, err := tree.File(path)
	if err != nil {
		if errors.Is(err, object.ErrFileNotFound) {
			return fmt.Errorf("path %q not found at HEAD: %w", path, err)
		}
		return fmt.Errorf("file: %w", err)
	}

	contents, err := file.Contents()
	if err != nil {
		return fmt.Errorf("contents: %w", err)
	}

	wt, err := r.r.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	f, err := wt.Filesystem.Create(path)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()

	_, err = io.WriteString(f, contents)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}
