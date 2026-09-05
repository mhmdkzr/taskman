// Package send provides the telegram_send tool backed by the Telegram Bot
// API. Build a Client in the parent telegram package, then pass it to Tool.
package send

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"

	"github.com/mhmdkzr/loop/internal/agent/tools/telegram"
)

type output struct {
	Sent      bool  `json:"sent"`
	ChatID    int64 `json:"chat_id"`
	MessageID int   `json:"message_id"`
}

func execute(ctx context.Context, c *telegram.Client, in input) (output, error) {
	message := strings.TrimSpace(in.Message)
	if message == "" {
		return output{}, fmt.Errorf("telegram_send: message is required")
	}
	if !c.Configured() {
		return output{}, fmt.Errorf("telegram_send: not configured: set TELEGRAM_API_KEY and TELEGRAM_CHANNEL_ID")
	}

	client, err := c.NewBot(c.APIKey)
	if err != nil {
		return output{}, fmt.Errorf("telegram_send: create bot: %w", err)
	}

	sent, err := client.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: c.ChannelID,
		Text:   message,
	})
	if err != nil {
		return output{}, fmt.Errorf("telegram_send: send message: %w", err)
	}

	messageID := 0
	if sent != nil {
		messageID = sent.ID
	}
	return output{Sent: true, ChatID: c.ChannelID, MessageID: messageID}, nil
}
