package read

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "read_file"
	description = "Read a file or list a directory from the local filesystem. Directory listings exclude images and PDFs."
)

const Description = description

type input struct {
	Path   string `json:"path"             jsonschema:"description=Path to the file to read."`
	Offset int    `json:"offset,omitempty" jsonschema:"description=1-indexed line number to start reading from. Defaults to 1."`
	Limit  int    `json:"limit,omitempty"  jsonschema:"description=Maximum number of lines to return. Defaults to 2000."`
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
	return nil
}
