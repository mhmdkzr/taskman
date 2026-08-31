# `app`

Application — the shared runtime context that every slice depends on. Provides the central `App` struct. Configuration lives in `internal/config`.

HTTP response helpers formerly here have been extracted to `pkg/jsonresp` (`jsonresp.WriteJSON`, `jsonresp.WriteHTTPError`). JetStream publishing is now in `pkg/produce` (`produce.Produce` with `MsgID`-based deduplication).

## Types

| Type | Description |
|---|---|
| `App` | Central runtime context — holds DB (`*sql.DB`), NATS `*nats.Conn`, JetStream `jetstream.JetStream`, Temporal client, TigerBeetle `tigerbeetle.Client`, Zitadel `*client.Client`, SMTP `*mail.Client`, and config (`config.Config`) |
| `Deps` | Shared runtime dependencies (`DB`, `NC`, `JS`, `Temporal`, `TigerBeetle`, `Zitadel`, `Mailer`) — see `pkg/stack` for anchored deps (`tigerbeetle-go`, `zitadel-go`, `go-mail`) |
| `Route` | HTTP route descriptor — method, path, handler function (see `internal/routes`). Registered with base-path prefixing via `App.Cfg.Server.BasePath` |
