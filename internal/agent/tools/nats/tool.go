package nats

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "nats_cli"
	Description = "Run the system nats CLI with the supplied arguments and return its standard output."
)

type Input struct {
	Args []string `json:"args" jsonschema:"description=Arguments to pass to nats, for example [\"stream\", \"ls\"]. Do not include the nats executable name."`
}

func Tool() goai.Tool {
	return tools.Tool(Name, Description, execute)
}

func (Input) Validate() error { return nil }
