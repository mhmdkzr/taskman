# `app`

Application — the shared runtime context that every slice depends on. Provides the central `App` struct. Configuration lives in `internal/app/config`.

HTTP response helpers formerly here have been extracted to `pkg/jsonresp` (`jsonresp.WriteJSON`, `jsonresp.WriteHTTPError`).

## Types

| Type | Description |
|---|---|
| `App` | Central runtime context — holds DB (`*sql.DB`) and config (`config.Config`) |
| `Deps` | Shared runtime dependencies (`DB`, `NC`, `JS`, `Temporal`, `TigerBeetle`, `Zitadel`, `Mailer`) — see `pkg/stack` for anchored deps (`tigerbeetle-go`, `zitadel-go`, `go-mail`) |
| `Route` | HTTP route descriptor — method, path, handler function (see `internal/app/routes`). Registered with base-path prefixing via `App.Cfg.Server.BasePath` |
