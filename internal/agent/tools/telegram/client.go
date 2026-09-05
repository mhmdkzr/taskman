// Package telegram provides the shared Client backing the telegram_send and
// telegram_read tools. Each tool lives in its own subpackage and takes the
// Client as a dependency.
package telegram

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/mhmdkzr/loop/internal/app/config"
)

const defaultBaseURL = "https://api.telegram.org"

// Bot is the minimal Telegram client surface used by the telegram_send tool.
type Bot interface {
	SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error)
}

// Client holds the credentials and dependencies shared by the telegram tools.
// BaseURL, HTTP, and NewBot are exposed so tests can point the tools at a fake
// server or bot.
type Client struct {
	APIKey    string
	ChannelID int64
	BaseURL   string
	HTTP      *http.Client
	NewBot    func(token string) (Bot, error)
}

// NewClient returns a Client for the given credentials. Empty values disable
// the tools; their execution then fails until the Client has credentials.
func NewClient(key string, id int64) *Client {
	return &Client{
		APIKey:    key,
		ChannelID: id,
		BaseURL:   defaultBaseURL,
		HTTP:      &http.Client{Timeout: 30 * time.Second},
		NewBot:    func(token string) (Bot, error) { return bot.New(token, bot.WithSkipGetMe()) },
	}
}

// NewClientFromConfig returns a Client from the given config, parsing the
// channel id. Empty credentials disable the tools.
func NewClientFromConfig(cfg config.TelegramConfig) (*Client, error) {
	var id int64
	if cfg.ChannelID != "" {
		parsed, err := strconv.ParseInt(cfg.ChannelID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid telegram channel id %q: %w", cfg.ChannelID, err)
		}
		id = parsed
	}
	return NewClient(cfg.APIKey, id), nil
}

// Configured reports whether credentials for the tools are present.
func (c *Client) Configured() bool {
	return c != nil && c.APIKey != "" && c.ChannelID != 0
}
