package read

import (
	"context"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/agent/tools/telegram"
)

const (
	Name        = "telegram_read"
	description = "Read the most recent text messages posted in the configured Telegram channel. " +
		"The bot must be an administrator of the channel. Read messages are acknowledged and will not be returned again."
)

const Description = description

type input struct {
	Limit *int `json:"limit,omitempty" jsonschema:"description=Maximum number of messages to return (default 10)."`
}

type Input = input
type Output = output

// Tool returns the telegram_read tool bound to c.
func Tool(c *telegram.Client) goai.Tool {
	return tools.Tool(Name, description, func(ctx context.Context, in input) (output, error) {
		return execute(ctx, c, in)
	})
}

func (input) Validate() error { return nil }
