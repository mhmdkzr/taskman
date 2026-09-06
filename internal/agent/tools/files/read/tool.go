package read

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "read_file"
	Description = "Read a file or list a directory from the local filesystem. Directory listings exclude images and PDFs."
)

type Input struct {
	Path   string `json:"path"             jsonschema:"description=Path to the file to read."`
	Offset int    `json:"offset,omitempty" jsonschema:"description=1-indexed line number to start reading from. Defaults to 1."`
	Limit  int    `json:"limit,omitempty"  jsonschema:"description=Maximum number of lines to return. Defaults to 2000."`
}

func Tool() goai.Tool {
	return tools.Tool(Name, Description, execute)
}

func (in Input) Validate() error {
	if in.Path == "" {
		return errPathRequired
	}
	return nil
}
