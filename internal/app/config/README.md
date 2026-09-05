# `config`

Application configuration loaded from environment variables.

## Architecture

Configuration is loaded entirely from environment variables using `github.com/caarlos0/env/v11`. An optional `.env` file can be loaded from the current working directory (skipped if `SKIP_ENV_AUTO_LOAD=true` is set). All fields are required — every configuration value must be provided via env var or `.env` file.

## Loading

The `Config.Load()` method:
1. Optionally loads `.env` from the working directory (skipped if `SKIP_ENV_AUTO_LOAD=true`)
2. Parses all environment variables into the `Config` struct using `github.com/caarlos0/env/v11`

## Types

| Type | Description |
|---|---|
| `Config` | Top-level application configuration |
| `ServerConfig` | HTTP server bind address and base path |
| `TemporalConfig` | Temporal host, namespace and task queue (`TEMPORAL_*`) |
| `TigerBeetleConfig` | TigerBeetle address and cluster ID (`TIGERBEETLE_*`) |
| `ZitadelConfig` | Zitadel client domain and insecure flag (`ZITADEL_CLIENT_*`) |
| `SMTPConfig` | SMTP mailer host/port/credentials (`SMTP_*`, MailHog defaults) |
| `TelegramConfig` | Telegram bot credentials for the agent tools (`TELEGRAM_*`) |
| `TavilyConfig` | Tavily API key for the agent websearch tool (`TAVILY_*`) |

Nested configuration types are defined in their respective packages:
- `pkg/logger.Config`
- application-owned SQLite configuration

### TigerBeetle

`TIGERBEETLE_ADDRESS` (default `127.0.0.1:3000`) and `TIGERBEETLE_CLUSTER_ID` (default `0`, `uint64`). Validated: `ADDRESS` non-empty. Compose override `tigerbeetle:3000`.

### Zitadel Client

`ZITADEL_CLIENT_DOMAIN` (default `127.0.0.1:8080`, e.g. `zitadel:8080` in compose) and `ZITADEL_CLIENT_INSECURE` (default `true`). Validated: `DOMAIN` non-empty. Built with `zitadel.New(domain, zitadel.WithInsecure(...))` and `client.New`.

### BFF authentication and private webhooks

`AUTH_*` configures the confidential server-side Zitadel OIDC client and the
SQLite-backed session. Authentication is always enabled.
`AUTH_INTERNAL_ADDRESS` is optional and Compose-only: it dials the private
provider address while retaining the public `AUTH_ISSUER` URL and Host header.
Issuer, client credentials, exact callback URLs, and positive session durations are
required. `WEBHOOKS_ZITADEL_PATH_SECRET` enables the private-network
notification handler on the existing app listener; Caddy deliberately returns 404 for its
public `/webhooks/*` counterpart.

### SMTP (MailHog)

`SMTP_HOST` (default `127.0.0.1`, compose `mailhog`), `SMTP_PORT` (default `1025`), `SMTP_FROM` (default `noreply@example.com`), `SMTP_FROM_NAME`, `SMTP_USERNAME`, `SMTP_PASSWORD`. Validated: `HOST` non-empty, `PORT` 1–65535, `FROM` contains `@`. Built with `mail.NewClient(host, mail.WithPort(port), ...)`. MailHog UI at `http://localhost:8025`, SMTP at `localhost:1025`.

### Agent tools

`TELEGRAM_API_KEY` (bot token), `TELEGRAM_CHANNEL_ID`, and `TAVILY_API_KEY` configure the agent's `telegram_send`/`telegram_read` and `websearch` tools. All are required: `Config.Load` fails when any is missing, and `validate` rejects empty values. Built with `telegram.NewClientFromConfig` and `websearch.NewClientFromConfig`.

### Model provider

`PROVIDER_BASE_URL`, `PROVIDER_API_KEY_OPENCODE`, `PROVIDER_MODEL`, and
`PROVIDER_REASONING_EFFORT` configure the current OpenCode provider. The API
key itself remains in the environment; the database stores only its environment
variable name.
