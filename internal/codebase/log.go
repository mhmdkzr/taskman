package codebase

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// LogOptions controls the commit log traversal.
type LogOptions struct {
	// MaxCount limits the number of commits to return.
	MaxCount int

	// Since includes commits after this time (exclusive).
	Since *time.Time

	// Until includes commits before this time (exclusive).
	Until *time.Time

	// Path restricts the log to commits that touch the given file path.
	Path string
}

type Commit struct {
	Hash      string
	Author    string
	Email     string
	When      time.Time
	Message   string
	ShortHash string
}

func (c Commit) Subject() string {
	for i := 0; i < len(c.Message); i++ {
		if c.Message[i] == '\n' {
			return c.Message[:i]
		}
	}
	return c.Message
}

func (r Repository) Log(opts LogOptions) ([]Commit, error) {
	head, err := r.r.Head()
	if err != nil {
		return nil, fmt.Errorf("head: %w", err)
	}

	logOpts := &git.LogOptions{
		From:  head.Hash(),
		Order: git.LogOrderCommitterTime,
	}
	if opts.MaxCount > 0 {
		logOpts.All = true
	}

	iter, err := r.r.Log(logOpts)
	if err != nil {
		return nil, fmt.Errorf("log: %w", err)
	}

	var commits []Commit
	err = iter.ForEach(func(c *object.Commit) error {
		if opts.MaxCount > 0 && len(commits) >= opts.MaxCount {
			return fmt.Errorf("stop")
		}
		if opts.Since != nil && c.Committer.When.Before(*opts.Since) {
			if len(commits) > 0 && opts.MaxCount == 0 {
				return fmt.Errorf("stop")
			}
			return nil
		}
		if opts.Until != nil && c.Committer.When.After(*opts.Until) {
			return nil
		}
		if opts.Path != "" {
			stats, err := c.Stats()
			if err != nil {
				return err
			}
			matched := false
			for _, s := range stats {
				if s.Name == opts.Path {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}

		h := c.Hash.String()
		commits = append(commits, Commit{
			Hash:      h,
			Author:    c.Author.Name,
			Email:     c.Author.Email,
			When:      c.Committer.When,
			Message:   c.Message,
			ShortHash: h[:7],
		})
		return nil
	})
	if err != nil && err.Error() == "stop" {
		err = nil
	}

	return commits, err
}
