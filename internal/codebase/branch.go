package codebase

import (
	"github.com/go-git/go-git/v5/plumbing"
)

type Branch struct {
	Name        string
	Remote      string
	Merge       plumbing.ReferenceName
	Rebase      string
	Description string
}

func (r Repository) Branch() (Branch, error) {
	head, err := r.r.Head()
	if err != nil {
		return Branch{}, err
	}
	if !head.Name().IsBranch() {
		return Branch{}, nil
	}
	name := head.Name().Short()
	cfg, err := r.r.Branch(name)
	if err != nil {
		return Branch{Name: name}, nil
	}
	return Branch{
		Name:        cfg.Name,
		Remote:      cfg.Remote,
		Merge:       cfg.Merge,
		Rebase:      cfg.Rebase,
		Description: cfg.Description,
	}, nil
}

func (r Repository) Branches() ([]Branch, error) {
	cfg, err := r.r.Config()
	if err != nil {
		return nil, err
	}
	var branches []Branch
	for _, b := range cfg.Branches {
		branches = append(branches, Branch{
			Name:        b.Name,
			Remote:      b.Remote,
			Merge:       b.Merge,
			Rebase:      b.Rebase,
			Description: b.Description,
		})
	}
	return branches, nil
}
