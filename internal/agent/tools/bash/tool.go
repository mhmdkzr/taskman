package bash

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "bash"
	Description = "Run a command through the system bash shell and return its standard output, standard error, and exit code."
)

type Input struct {
	Command string `json:"command" jsonschema:"description=Shell command to execute with bash."`
}

func Tool() goai.Tool {
	return tools.Tool(Name, Description, execute)
}

func (in Input) Validate() error {
	if in.Command == "" {
		return ErrEmptyCommand
	}
	return nil
}
