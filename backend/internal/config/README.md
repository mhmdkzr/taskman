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
| `TemporalConfig` | Temporal host and namespace |

Nested configuration types are defined in their respective packages:
- `pkg/logger.Config`
- `pkg/pg.Config`
- `pkg/notifier.Config`
