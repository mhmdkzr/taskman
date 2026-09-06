// Package patch implements OpenCode-compatible multi-file patch application.
package patch

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "apply_patch"
	Description = "Apply a multi-file patch. Supports adding, updating, deleting, and moving files using the OpenCode patch format."
)

type Input struct {
	PatchText string `json:"patchText" jsonschema:"description=The full patch text that describes all changes to be made."`
}

type Output struct {
	Added    []string `json:"added,omitempty"`
	Modified []string `json:"modified,omitempty"`
	Deleted  []string `json:"deleted,omitempty"`
}

func Tool() goai.Tool {
	return tools.Tool(Name, Description, execute)
}

func (in Input) Validate() error {
	if in.PatchText == "" {
		return errPatchRequired
	}
	return nil
}
