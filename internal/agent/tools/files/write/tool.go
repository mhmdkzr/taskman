package write

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "write_file"
	description = "Create a new file with the given content, or overwrite an existing file's entire contents. Does not create missing parent directories."
)

const Description = description

type input struct {
	Path    string `json:"path"    jsonschema:"description=Path to the file to create or overwrite."`
	Content string `json:"content" jsonschema:"description=Content to write to the file."`
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
