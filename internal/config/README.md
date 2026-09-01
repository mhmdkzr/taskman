# `config`

Application configuration loaded from environment variables.

## Architecture

Configuration is loaded entirely from environment variables using `github.com/caarlos0/env/v11`. An optional `.env` file can be loaded from the current working directory (skipped if `SKIP_ENV_AUTO_LOAD=true` is set). Fields without a default are required — every configuration value must be provided via env var or `.env` file.

## Loading

The `Config.Load()` method:
1. Optionally loads `.env` from the working directory (skipped if `SKIP_ENV_AUTO_LOAD=true`)
2. Parses all environment variables into the `Config` struct using `github.com/caarlos0/env/v11`

`Config.Validate()` then checks each section, and `Config.AgentOptions()` returns the normalized agent configuration (defaults applied, `~` expanded in the DB path).

## Types

| Type | Description |
|---|---|
| `Config` | Top-level application configuration |
| `NATSConfig` | NATS server URL (`NATS_*`) |
| `ServerConfig` | HTTP server bind address, base path and timeouts (`SERVER_*`) |
| `AgentConfig` | Agent runtime: provider credentials, DB path, defaults and telegram tool (`AGENT_*`) |
| `ProviderConfig` | LLM provider base URL and API key (`AGENT_PROVIDER_*`) |
| `Telegram` | telegram_send / telegram_read credentials (`AGENT_TELEGRAM_*`) |

Nested configuration types are defined in their respective packages:
- `pkg/logger.Config`