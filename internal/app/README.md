# `app`

Application core — the shared runtime context that every slice depends on. Provides the central `App` struct and NATS/JetStream abstractions. Configuration lives in `internal/config`.

HTTP response helpers formerly here have been extracted to `pkg/resp` (`resp.WriteJSON`, `resp.WriteHTTPError`).

## Types

| Type | Description |
|---|---|
| `App` | Central runtime context — holds DB (`*sql.DB`), NATS `*nats.Conn`, JetStream `jetstream.JetStream`, Temporal client, and config (`config.Config`) |
| `Route` | HTTP route descriptor — method, path, handler function. Registered with base-path prefixing |
## NATS / JetStream

| Function | Description |
|---|---|
| `Produce(ctx, js, subject, event)` | Publish a typed event to JetStream with `MsgID`-based deduplication |
