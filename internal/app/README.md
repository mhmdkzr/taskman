# `app`

Application — the shared runtime context that every slice depends on. Provides the central `App` struct. Configuration lives in `internal/config`.

HTTP response helpers formerly here have been extracted to `pkg/jsonresp` (`jsonresp.WriteJSON`, `jsonresp.WriteHTTPError`). JetStream publishing is now in `pkg/produce` (`produce.Produce` with `MsgID`-based deduplication).

## Types

| Type | Description |
|---|---|
| `App` | Central runtime context — holds NATS `*nats.Conn`, JetStream `jetstream.JetStream`, and config (`config.Config`) |
| `Deps` | Shared runtime dependencies (`NC`, `JS`) |
