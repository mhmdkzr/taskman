# `pkg/logger`

Application-wide structured logging via `slog`, with text or JSON output and an
optional NATS handler that publishes warn/error logs for remote aggregation.

## API

- `Init(cfg Config, nc *nats.Conn) error` — configures the default slog logger.
  Pass a nil `*nats.Conn` to disable NATS integration. Output goes to stdout.
- `Config{Format LogFormat, Level slog.Level}` — loaded from environment
  variables (`FORMAT`: `"json"` or `"text"`; `LEVEL`: a slog level). `Validate`
  rejects invalid values.
- `NewNATSHandler(wrapped slog.Handler, nc *nats.Conn) *NATSHandler` — wraps a
  base handler; usable directly if needed.

## NATS behavior

When a NATS connection is provided, records at Warn level or above are also
published as JSON (`timestamp`, `level`, `message`, `attributes`) to
level-specific subjects:

| Level | Subject |
|---|---|
| Debug | `system.logs.debug` |
| Info | `system.logs.info` |
| Warn | `system.logs.warn` |
| Error | `system.logs.error` |

Attributes include grouped keys (flattened with dot notation), source location
(`source.file`, `source.line`, optional `function`), and errors serialized by
their message string. Sensitive attribute keys are redacted via
`IsSensitiveKey` and replaced with the `Marker` before publishing.

Failures inside the handler itself (marshal, publish) never
break logging: they are logged at Error level through the wrapped handler.

## Usage

```go
cfg := logger.Config{Format: logger.LogFormatJSON, Level: slog.LevelInfo}
if err := cfg.Validate(); err != nil { ... }
if err := logger.Init(cfg, nc); err != nil { ... }

// Anywhere else in the app:
slog.Error("failed to broadcast tx", "error", err)
```
