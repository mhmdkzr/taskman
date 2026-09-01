package codebase

import (
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/pmezard/go-difflib/difflib"
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

// unifiedDiffContextLines is the number of unchanged lines shown around each
// hunk, matching `git diff`'s own default.
const unifiedDiffContextLines = 3

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

		patchText, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(oldContent),
			B:        difflib.SplitLines(newContent),
			FromFile: "a/" + path,
			ToFile:   "b/" + path,
			Context:  unifiedDiffContextLines,
		})
		if err != nil {
			return nil, fmt.Errorf("diff %s: %w", path, err)
		}

		additions, deletions := countUnifiedDiffLines(patchText)

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

// countUnifiedDiffLines counts added/removed content lines in a unified diff
// produced by difflib.GetUnifiedDiffString, ignoring the "--- a/..."/"+++
// b/..." file headers those lines' own "-"/"+" prefixes would otherwise be
// mistaken for.
func countUnifiedDiffLines(patch string) (int, int) {
	var additions, deletions int
	for line := range strings.SplitSeq(patch, "\n") {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
		case strings.HasPrefix(line, "+"):
			additions++
		case strings.HasPrefix(line, "-"):
			deletions++
		}
	}
	return additions, deletions
}
