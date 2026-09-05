package send

import (
	"context"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/agent/tools/telegram"
)

const (
	Name        = "telegram_send"
	description = "Send a plain text Telegram message to the configured channel."
)

const Description = description

type input struct {
	Message string `json:"message" jsonschema:"description=The plain text message to send."`
}

type Input = input
type Output = output

// Tool returns the telegram_send tool bound to c.
func Tool(c *telegram.Client) goai.Tool {
	return tools.Tool(Name, description, func(ctx context.Context, in input) (output, error) {
		return execute(ctx, c, in)
	})
}

func (input) Validate() error { return nil }
