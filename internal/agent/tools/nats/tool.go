package nats

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "nats_cli"
	description = "Run the system nats CLI with the supplied arguments and return its standard output."
)

const Description = description

type input struct {
	Args []string `json:"args" jsonschema:"description=Arguments to pass to nats, for example [\"stream\", \"ls\"]. Do not include the nats executable name."`
}

type Input = input

func Tool() goai.Tool {
	return tools.Tool(Name, description, execute)
}

func (input) Validate() error { return nil }
