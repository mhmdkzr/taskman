package codebase

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type CommitMessage struct {
	msg string
}

func NewCommitMessage(msg string) CommitMessage {
	return CommitMessage{msg: msg}
}

func (m CommitMessage) String() string {
	return m.msg
}

type AuthorSignature struct {
	Name  string
	Email string
}

type CommitResult struct {
	Hash  CommitHash
	Diffs []Diff
}

type CommitHash string

func NewCommitHash(hash string) (CommitHash, error) {
	return CommitHash(hash), nil
}

func (r Repository) Stage(paths ...string) error {
	wt, err := r.r.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}
	for _, path := range paths {
		if _, err := wt.Add(path); err != nil {
			return fmt.Errorf("stage %s: %w", path, err)
		}
	}
	return nil
}

func (r Repository) Commit(msg CommitMessage, sig AuthorSignature, paths ...string) (*CommitResult, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("status: %w", err)
	}
	if len(status) == 0 {
		return nil, fmt.Errorf("nothing to commit, working tree is clean")
	}

	if len(paths) > 0 {
		if err := r.Stage(paths...); err != nil {
			return nil, err
		}
	} else {
		if err := wt.AddGlob("."); err != nil {
			return nil, fmt.Errorf("add: %w", err)
		}
	}

	author := &object.Signature{Name: sig.Name, Email: sig.Email}
	if cfg, err := r.r.Config(); err == nil && cfg != nil {
		if cfg.User.Name != "" {
			author.Name = cfg.User.Name
		}
		if cfg.User.Email != "" {
			author.Email = cfg.User.Email
		}
	}

	hash, err := wt.Commit(msg.String(), &git.CommitOptions{
		Author: author,
	})
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	diffs, err := r.Diff()
	if err != nil {
		diffs = nil
	}

	h, err := NewCommitHash(hash.String())
	if err != nil {
		return nil, fmt.Errorf("new commit hash: %w", err)
	}

	return &CommitResult{Hash: h, Diffs: diffs}, nil
}
