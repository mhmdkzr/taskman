// Package patch implements OpenCode-compatible multi-file patch application.
package patch

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "apply_patch"
	description = "Apply a multi-file patch. Supports adding, updating, deleting, and moving files using the OpenCode patch format."
)

const Description = description

type input struct {
	PatchText string `json:"patchText" jsonschema:"description=The full patch text that describes all changes to be made."`
}

type output struct {
	Added    []string `json:"added,omitempty"`
	Modified []string `json:"modified,omitempty"`
	Deleted  []string `json:"deleted,omitempty"`
}

type Input = input
type Output = output

func Tool() goai.Tool {
	return tools.Tool(Name, description, execute)
}

func (in input) Validate() error {
	if in.PatchText == "" {
		return errPatchRequired
	}
	return nil
}
