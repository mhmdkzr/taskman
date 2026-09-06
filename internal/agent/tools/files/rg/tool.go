package rg

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "rg_cli"
	Description = "Run the system ripgrep (rg) CLI with the supplied arguments and return its standard output."
)

type Input struct {
	Args []string `json:"args" jsonschema:"description=Arguments to pass to rg. Do not include the rg executable name."`
}

func Tool() goai.Tool {
	return tools.Tool(Name, Description, execute)
}

func (Input) Validate() error { return nil }
