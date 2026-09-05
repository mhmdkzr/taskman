package edit

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "edit_file"
	description = "Replace an exact string match in a file. Fails if the old string is not found, or if it matches more than once and replace_all is not set."
)

const Description = description

type input struct {
	Path       string `json:"path"                  jsonschema:"description=Path to the file to edit."`
	OldString  string `json:"old_string"            jsonschema:"description=Exact text to search for. Must match the file contents exactly, including whitespace."`
	NewString  string `json:"new_string"            jsonschema:"description=Text to replace old_string with."`
	ReplaceAll bool   `json:"replace_all,omitempty" jsonschema:"description=Replace every occurrence of old_string instead of requiring a single unique match. Defaults to false."`
}

type (
	Input  = input
	Output = output
)

func Tool() goai.Tool {
	return tools.Tool(Name, description, execute)
}

func (in input) Validate() error {
	if in.Path == "" {
		return errPathRequired
	}
	if in.OldString == "" {
		return errOldStringRequired
	}
	if in.OldString == in.NewString {
		return errNoChange
	}
	return nil
}
