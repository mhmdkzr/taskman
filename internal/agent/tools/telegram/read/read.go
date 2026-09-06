// Package read provides the telegram_read tool backed by the Telegram Bot
// API. Build a Client in the parent telegram package, then pass it to Tool.
//
// Messages are read via short-polling getUpdates and acknowledged so they are
// not returned again by later calls.
package read

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mhmdkzr/loop/internal/agent/tools/telegram"
)

const (
	defaultLimit = 10
	maxLimit     = 100
)

func execute(ctx context.Context, c *telegram.Client, in Input) (Output, error) {
	if !c.Configured() {
		return Output{}, fmt.Errorf("telegram_read: not configured: set TELEGRAM_API_KEY and TELEGRAM_CHANNEL_ID")
	}

	limit := defaultLimit
	if in.Limit != nil {
		limit = *in.Limit
	}
	if limit < 1 || limit > maxLimit {
		return Output{}, fmt.Errorf("telegram_read: limit must be between 1 and %d", maxLimit)
	}

	updates, err := fetchUpdates(ctx, c.BaseURL, c.HTTP, c.APIKey)
	if err != nil {
		return Output{}, fmt.Errorf("telegram_read: %w", err)
	}

	var msgs []message
	maxUpdateID := int64(0)
	for _, u := range updates {
		if u.UpdateID > maxUpdateID {
			maxUpdateID = u.UpdateID
		}
		if u.Message != nil && u.Message.Chat != nil && u.Message.Chat.ID == c.ChannelID {
			msgs = append(msgs, *u.Message)
		}
		if u.ChannelPost != nil && u.ChannelPost.Chat != nil && u.ChannelPost.Chat.ID == c.ChannelID {
			msgs = append(msgs, *u.ChannelPost)
		}
	}

	if maxUpdateID > 0 {
		if err := ackUpdates(ctx, c.BaseURL, c.HTTP, c.APIKey, maxUpdateID); err != nil {
			return Output{}, fmt.Errorf("telegram_read: ack updates: %w", err)
		}
	}

	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}

	result := Output{Messages: make([]OutputMessage, 0, len(msgs))}
	for _, m := range msgs {
		text := strings.TrimSpace(m.Text)
		if text == "" {
			text = "[media message]"
		}
		result.Messages = append(result.Messages, OutputMessage{
			Sender: senderName(m.From),
			Date:   time.Unix(m.Date, 0).UTC().Format("2006-01-02 15:04"),
			Text:   text,
		})
	}
	return result, nil
}

type chat struct {
	ID int64 `json:"id"`
}

type user struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type message struct {
	Chat *chat  `json:"chat"`
	From *user  `json:"from"`
	Date int64  `json:"date"`
	Text string `json:"text"`
}

type OutputMessage struct {
	Sender string `json:"sender"`
	Date   string `json:"date"`
	Text   string `json:"text"`
}

// Output is the result returned by telegram_read.
type Output struct {
	Messages []OutputMessage `json:"messages"`
}

type update struct {
	UpdateID    int64    `json:"update_id"`
	Message     *message `json:"message"`
	ChannelPost *message `json:"channel_post"`
}

func senderName(u *user) string {
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

// fetchUpdates retrieves up to maxLimit pending updates via short polling,
// limited to message and channel_post update types.
func fetchUpdates(ctx context.Context, baseURL string, client *http.Client, token string) ([]update, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(maxLimit))
	params.Set("timeout", "0")
	params.Set("allowed_updates", `["message","channel_post"]`)

	endpoint := baseURL + "/bot" + token + "/getUpdates?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build getUpdates request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call getUpdates: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("close getUpdates response body", "error", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read getUpdates response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getUpdates status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		OK     bool     `json:"ok"`
		Result []update `json:"result"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload.Result, nil
}

// ackUpdates confirms all updates up to maxUpdateID so they are not returned
// again by later getUpdates calls.
func ackUpdates(ctx context.Context, baseURL string, client *http.Client, token string, maxUpdateID int64) error {
	params := url.Values{}
	params.Set("offset", strconv.FormatInt(maxUpdateID+1, 10))
	params.Set("limit", "1")
	params.Set("timeout", "0")

	endpoint := baseURL + "/bot" + token + "/getUpdates?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build getUpdates ack request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("call getUpdates ack: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("close getUpdates ack response body", "error", err)
		}
	}()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("drain getUpdates ack response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("getUpdates status %d", resp.StatusCode)
	}
	return nil
}
