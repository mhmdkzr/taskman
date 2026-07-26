# `app`

Application core — the shared runtime context that every slice depends on. Provides the central `App` struct, HTTP utilities, and NATS/JetStream abstractions. Configuration lives in `internal/config`.

## Types

| Type | Description |
|---|---|
| `App` | Central runtime context — holds DB (`*sql.DB`), NATS `*nats.Conn`, JetStream `jetstream.JetStream`, Temporal client, and config (`config.Config`) |
| `Route` | HTTP route descriptor — method, path, handler function. Registered with base-path prefixing |
| `ErrorResponse` | JSON error format used for HTTP error responses |

## HTTP Utilities

| Function | Description |
|---|---|
| `WriteJSON(w, status, v)` | Serializes `v` as JSON and writes it with the given HTTP status |
| `WriteHTTPError(w, status, err)` | Writes a JSON-error response with the given HTTP status |

## NATS / JetStream

| Function | Description |
|---|---|
| `Produce(ctx, js, subject, event)` | Publish a typed event to JetStream with `MsgID`-based deduplication |
