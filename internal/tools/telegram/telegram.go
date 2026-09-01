// Package telegram provides telegram_send and telegram_read tools backed by
// the Telegram Bot API. Build a Client with the credentials, then pass it to
// SendTool and ReadTool.
package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/zendev-sh/goai"
)

const (
	defaultTelegramReadLimit = 10
	maxTelegramFetchLimit    = 100
	defaultTelegramBaseURL   = "https://api.telegram.org"
)

// Client holds the credentials and dependencies shared by the telegram tools.
type Client struct {
	apiKey    string
	channelID int64
	baseURL   string
	http      *http.Client
	newBot    telegramBotFactory
	fetch     func(baseURL string, client *http.Client, token string) ([]telegramUpdate, error)
	ack       func(baseURL string, client *http.Client, token string, maxID int64) error
}

// NewClient returns a Client for the given credentials. Empty values disable
// the tools; their execution then fails until the Client has credentials.
func NewClient(key string, id int64) *Client {
	return &Client{
		apiKey:    key,
		channelID: id,
		baseURL:   defaultTelegramBaseURL,
		http:      &http.Client{Timeout: 30 * time.Second},
		newBot:    defaultTelegramBotFactory,
		fetch:     fetchTelegramUpdates,
		ack:       ackTelegramUpdates,
	}
}

// NewClientFromConfig returns a Client from the given config, parsing the
// channel id. Empty credentials disable the tools.
func NewClientFromConfig(cfg config.Telegram) (*Client, error) {
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
	return c != nil && c.apiKey != "" && c.channelID != 0
}

type sendInput struct {
	Message string `json:"message" jsonschema:"description=The plain text message to send."`
}

// SendTool returns the telegram_send tool.
func SendTool(c *Client) goai.Tool {
	return goai.NewTool("telegram_send",
		"Send a plain text Telegram message to the configured channel.",
		func(ctx context.Context, in sendInput) (string, error) {
			message := strings.TrimSpace(in.Message)
			if message == "" {
				return "", fmt.Errorf("telegram_send: message is required")
			}
			if !c.Configured() {
				return "", fmt.Errorf("telegram_send: not configured: set telegram.api_key and telegram.channel_id in config")
			}

			client, err := c.newBot(c.apiKey)
			if err != nil {
				return "", fmt.Errorf("telegram_send: create bot: %w", err)
			}

			sent, err := client.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: c.channelID,
				Text:   message,
			})
			if err != nil {
				return "", fmt.Errorf("telegram_send: send message: %w", err)
			}

			messageID := 0
			if sent != nil {
				messageID = sent.ID
			}
			out, _ := json.Marshal(map[string]any{
				"sent":       true,
				"chat_id":    c.channelID,
				"message_id": messageID,
			})
			return string(out), nil
		})
}

type readInput struct {
	Limit *int `json:"limit,omitempty" jsonschema:"description=Maximum number of messages to return (default 10)."`
}

// ReadTool returns the telegram_read tool.
func ReadTool(c *Client) goai.Tool {
	return goai.NewTool("telegram_read",
		"Read the most recent text messages posted in the configured Telegram channel. "+
			"The bot must be an administrator of the channel. Read messages are acknowledged and will not be returned again.",
		func(ctx context.Context, in readInput) (string, error) {
			if !c.Configured() {
				return "", fmt.Errorf("telegram_read: not configured: set telegram.api_key and telegram.channel_id in config")
			}

			limit := defaultTelegramReadLimit
			if in.Limit != nil {
				limit = *in.Limit
			}
			if limit < 1 || limit > maxTelegramFetchLimit {
				return "", fmt.Errorf("telegram_read: limit must be between 1 and %d", maxTelegramFetchLimit)
			}

			updates, err := c.fetch(c.baseURL, c.http, c.apiKey)
			if err != nil {
				return "", fmt.Errorf("telegram_read: %w", err)
			}

			var msgs []telegramMessage
			maxUpdateID := int64(0)
			for _, u := range updates {
				if u.UpdateID > maxUpdateID {
					maxUpdateID = u.UpdateID
				}
				if u.Message != nil && u.Message.Chat != nil && u.Message.Chat.ID == c.channelID {
					msgs = append(msgs, *u.Message)
				}
				if u.ChannelPost != nil && u.ChannelPost.Chat != nil && u.ChannelPost.Chat.ID == c.channelID {
					msgs = append(msgs, *u.ChannelPost)
				}
			}

			if maxUpdateID > 0 {
				if err := c.ack(c.baseURL, c.http, c.apiKey, maxUpdateID); err != nil {
					return "", fmt.Errorf("telegram_read: ack updates: %w", err)
				}
			}

			if len(msgs) == 0 {
				return "no new messages", nil
			}
			if len(msgs) > limit {
				msgs = msgs[len(msgs)-limit:]
			}

			var b strings.Builder
			for _, m := range msgs {
				text := strings.TrimSpace(m.Text)
				if text == "" {
					text = "[media message]"
				}
				b.WriteString(senderName(m.From))
				b.WriteString(" (")
				b.WriteString(time.Unix(m.Date, 0).UTC().Format("2006-01-02 15:04"))
				b.WriteString("): ")
				b.WriteString(text)
				b.WriteByte('\n')
			}
			return strings.TrimSuffix(b.String(), "\n"), nil
		})
}

// --- telegram_send plumbing ---

type telegramClient interface {
	SendMessage(context.Context, *bot.SendMessageParams) (*models.Message, error)
}

type telegramBotFactory func(token string) (telegramClient, error)

func defaultTelegramBotFactory(token string) (telegramClient, error) {
	return bot.New(token, bot.WithSkipGetMe())
}

// --- telegram_read plumbing ---

type telegramChat struct {
	ID int64 `json:"id"`
}

type telegramUser struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type telegramMessage struct {
	Chat *telegramChat `json:"chat"`
	From *telegramUser `json:"from"`
	Date int64         `json:"date"`
	Text string        `json:"text"`
}

type telegramUpdate struct {
	UpdateID    int64            `json:"update_id"`
	Message     *telegramMessage `json:"message"`
	ChannelPost *telegramMessage `json:"channel_post"`
}

func senderName(u *telegramUser) string {
	if u == nil {
		return "unknown"
	}
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name == "" {
		name = strings.TrimSpace(u.Username)
	}
	if name == "" {
		return "unknown"
	}
	return name
}

// fetchTelegramUpdates retrieves up to maxTelegramFetchLimit pending updates
// via short polling, limited to message and channel_post update types.
func fetchTelegramUpdates(baseURL string, client *http.Client, token string) ([]telegramUpdate, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(maxTelegramFetchLimit))
	params.Set("timeout", "0")
	params.Set("allowed_updates", `["message","channel_post"]`)

	resp, err := client.Get(baseURL + "/bot" + token + "/getUpdates?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getUpdates status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		OK     bool             `json:"ok"`
		Result []telegramUpdate `json:"result"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload.Result, nil
}

// ackTelegramUpdates confirms all updates up to maxUpdateID so they are not
// returned again by later getUpdates calls.
func ackTelegramUpdates(baseURL string, client *http.Client, token string, maxUpdateID int64) error {
	params := url.Values{}
	params.Set("offset", strconv.FormatInt(maxUpdateID+1, 10))
	params.Set("limit", "1")
	params.Set("timeout", "0")

	resp, err := client.Get(baseURL + "/bot" + token + "/getUpdates?" + params.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("getUpdates status %d", resp.StatusCode)
	}
	return nil
}
