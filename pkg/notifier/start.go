package notifier

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/mhmdkzr/app/pkg/logger"
)

var subjects = []string{
	logger.SubjectLogsError,
	logger.SubjectLogsWarn,
}

func Start(ctx context.Context, nc *nats.Conn, cfg Config) error {
	bot, err := initBot(cfg.Telegram.BotToken)
	if err != nil {
		return fmt.Errorf("failed to initialize bot: %w", err)
	}

	h := NewHandler(bot, cfg.Telegram.ChannelID)
	h.Start(ctx)

	var subs []*nats.Subscription
	for _, subject := range subjects {
		sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
			select {
			case h.MsgCh() <- msg:
			default:
				slog.Warn("notifier queue full, dropping message", "subject", subject)
			}
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe to NATS subject %s: %w", subject, err)
		}
		subs = append(subs, sub)
		slog.Info("Subscribed to NATS subject", "subject", subject, "channel", cfg.Telegram.ChannelID)
	}

	go func() {
		<-ctx.Done()
		for _, sub := range subs {
			if err := sub.Unsubscribe(); err != nil {
				slog.Error("failed to unsubscribe from NATS subject", "error", err)
			}
		}
	}()

	return nil
}
