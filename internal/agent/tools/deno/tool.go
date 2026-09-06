package deno

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "deno_cli"
	Description = "Run the system deno CLI with the supplied arguments and return its standard output."
)

type Input struct {
	Args []string `json:"args" jsonschema:"description=Arguments to pass to deno. Do not include the deno executable name."`
}

func Tool() goai.Tool {
	return tools.Tool(Name, Description, execute)
}

func (Input) Validate() error { return nil }
