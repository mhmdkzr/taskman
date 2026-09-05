package deno

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "deno_cli"
	description = "Run the system deno CLI with the supplied arguments and return its standard output."
)

const Description = description

type input struct {
	Args []string `json:"args" jsonschema:"description=Arguments to pass to deno. Do not include the deno executable name."`
}

type Input = input

func Tool() goai.Tool {
	return tools.Tool(Name, description, execute)
}

func (input) Validate() error { return nil }
