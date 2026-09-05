package temporal

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "temporal_cli"
	description = "Run the system temporal CLI with the supplied arguments and return its standard output."
)

const Description = description

type input struct {
	Args []string `json:"args" jsonschema:"description=Arguments to pass to temporal. Do not include the temporal executable name."`
}

type Input = input

func Tool() goai.Tool {
	return tools.Tool(Name, description, execute)
}

func (input) Validate() error { return nil }
