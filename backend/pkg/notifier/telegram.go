package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nats-io/nats.go"
)

const (
	telegramMaxMessageRunes = 4096
	truncationStepRunes     = 256
	batchInterval           = 2 * time.Second
	queueCapacity           = 64
)

type Handler struct {
	bot       *tgbotapi.BotAPI
	channelID int64
	msgCh     chan *nats.Msg
}

func NewHandler(bot *tgbotapi.BotAPI, channelID int64) *Handler {
	return &Handler{
		bot:       bot,
		channelID: channelID,
		msgCh:     make(chan *nats.Msg, queueCapacity),
	}
}

func (h *Handler) MsgCh() chan<- *nats.Msg {
	return h.msgCh
}

func (h *Handler) Start(ctx context.Context) {
	go h.processMessages(ctx)
}

func (h *Handler) processMessages(ctx context.Context) {
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	var batch []*nats.Msg
	flush := func() {
		if len(batch) == 0 {
			return
		}
		h.sendBatch(batch)
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case msg := <-h.msgCh:
			batch = append(batch, msg)
		case <-ticker.C:
			flush()
		}
	}
}

func (h *Handler) sendBatch(msgs []*nats.Msg) {
	message := formatBatchMessage(msgs)
	if err := h.sendMsg(message); err != nil {
		var apiErr tgbotapi.Error
		if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
			<-time.After(time.Duration(apiErr.RetryAfter) * time.Second)
			if err := h.sendMsg(message); err != nil {
				fmt.Fprintf(os.Stderr, "notifier: failed to send telegram message after retry: %v\n", err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "notifier: failed to send telegram message: %v\n", err)
		}
	}
}

func (h *Handler) sendMsg(message string) error {
	tgMsg := tgbotapi.NewMessage(h.channelID, message)
	tgMsg.ParseMode = tgbotapi.ModeHTML
	_, err := h.bot.Send(tgMsg)
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}
	return nil
}

func formatBatchMessage(msgs []*nats.Msg) string {
	parts := make([]string, 0, len(msgs))
	for _, msg := range msgs {
		parts = append(parts, formatNotificationMessage(msg.Subject, msg.Data))
	}

	body := strings.Join(parts, "\n\n---\n\n")
	if len(msgs) > 1 {
		body = fmt.Sprintf("[Batched notifications - %d messages]\n\n%s", len(msgs), body)
	}

	const suffix = "\n\n... (truncated)"
	for utf8.RuneCountInString(body) > telegramMaxMessageRunes {
		currentLen := utf8.RuneCountInString(body)
		if currentLen <= len(suffix)+1 {
			return "[Batched notifications]\n\nmessage too long"
		}
		nextLen := max(currentLen-truncationStepRunes, 1)
		body = truncateRunes(body, nextLen) + suffix
	}

	return body
}

func initBot(botToken string) (*tgbotapi.BotAPI, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}
	slog.Info("Authorized on Telegram account", "username", bot.Self.UserName)
	return bot, nil
}

func formatNotificationMessage(subject string, payload []byte) string {
	body := string(payload)
	var buf bytes.Buffer
	if err := json.Indent(&buf, payload, "", "  "); err == nil {
		body = buf.String()
	}

	return renderTelegramMessage(subject, body)
}

func renderTelegramMessage(subject string, body string) string {
	const truncationSuffix = "\n\n... (truncated)"
	for {
		msg := fmt.Sprintf("[%s]\n\n<pre>%s</pre>", html.EscapeString(subject), html.EscapeString(body))
		if utf8.RuneCountInString(msg) <= telegramMaxMessageRunes {
			return msg
		}

		currentLen := utf8.RuneCountInString(body)
		if currentLen <= len(truncationSuffix)+1 {
			return fmt.Sprintf("[%s]\n\n<pre>%s</pre>", html.EscapeString(subject), "message too long")
		}

		nextLen := max(currentLen-truncationStepRunes, 1)
		body = truncateRunes(body, nextLen) + truncationSuffix
	}
}

func truncateRunes(s string, maxRunes int) string {
	if maxRunes < 0 || utf8.RuneCountInString(s) <= maxRunes {
		return s
	}

	runes := []rune(s)
	return string(runes[:maxRunes])
}
