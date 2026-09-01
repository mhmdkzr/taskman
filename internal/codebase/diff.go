package codebase

import (
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type Diff struct {
	Name       string
	ChangeType ChangeType
	Additions  int
	Deletions  int
	Patch      string
}

type ChangeType string

const (
	ChangeTypeModified ChangeType = "M"
	ChangeTypeAdded    ChangeType = "A"
	ChangeTypeDeleted  ChangeType = "D"
)

func (r Repository) Diff() ([]Diff, error) {
	head, err := r.r.Head()
	if err != nil {
		return nil, fmt.Errorf("head: %w", err)
	}
	commit, err := r.r.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	headTree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("head tree: %w", err)
	}

	wt, err := r.r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("status: %w", err)
	}

	dmp := diffmatchpatch.New()
	var results []Diff

	for path, fs := range status {
		if fs.Worktree == git.Untracked {
			continue
		}

		var oldContent, newContent string

		if fs.Staging != git.Added {
			if f, err := headTree.File(path); err == nil {
				oldContent, _ = f.Contents()
			}
		}

		if fs.Worktree != git.Deleted {
			if f, err := wt.Filesystem.Open(path); err == nil {
				b, _ := io.ReadAll(f)
				newContent = string(b)
				f.Close()
			}
		}

		ct := ChangeTypeModified
		switch {
		case fs.Staging == git.Added || fs.Worktree == git.Added:
			ct = ChangeTypeAdded
		case fs.Staging == git.Deleted || fs.Worktree == git.Deleted:
			ct = ChangeTypeDeleted
		}

		diffs := dmp.DiffMain(oldContent, newContent, true)
		diffs = dmp.DiffCleanupSemantic(diffs)
		patches := dmp.PatchMake(diffs)
		patchText := dmp.PatchToText(patches)

		var additions, deletions int
		for _, d := range diffs {
			n := strings.Count(d.Text, "\n")
			if len(d.Text) > 0 && !strings.HasSuffix(d.Text, "\n") {
				n++
			}
			switch d.Type {
			case diffmatchpatch.DiffInsert:
				additions += n
			case diffmatchpatch.DiffDelete:
				deletions += n
			}
		}

		results = append(results, Diff{
			Name:       path,
			ChangeType: ct,
			Additions:  additions,
			Deletions:  deletions,
			Patch:      patchText,
		})
	}

	return results, nil
}
