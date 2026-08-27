# notifier

Operational alerting bridge: subscribes to the NATS log subjects `system.logs.error` and `system.logs.warn` (published by `pkg/logger`) and forwards each message as an HTML-formatted notification to a Telegram channel.

## Behavior

- Started as a background process with the NATS connection and config; one subscription per subject.
- Messages are handed to a worker via a bounded channel — if the queue is full, the message is dropped with a warning rather than blocking log delivery.
- Disabled cleanly when `ENABLED` is false.

## Configuration

Environment variables (see `Config`):

- `ENABLED` — toggles the notifier
- `TELEGRAM_BOT_TOKEN` — bot token (required when enabled)
- `TELEGRAM_CHANNEL_ID` — target channel ID (required when enabled)
