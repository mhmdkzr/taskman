# `config`

Application configuration loaded from environment variables with
`github.com/caarlos0/env/v11`.

## Loading

`Config.Load()` loads `.env` from the current working directory unless
`SKIP_ENV_AUTO_LOAD=true`, then parses the environment. `Config.LoadFrom(path)`
uses the supplied env file instead. Failure to load the env file is logged, but
parsing errors are returned to the caller. Validation errors are returned by
`Config.Validate()`.

Fields without defaults are required by the parser. Call `Config.Validate()`
after loading to apply section-specific validation; loading does not call
`Validate()` automatically.

## Configuration

| Section | Environment variables | Defaults |
|---|---|---|
| `ServerConfig` | `SERVER_BIND_ADDR`, `SERVER_BASE_PATH`, `SERVER_TIMEOUT`, `SERVER_SHUTDOWN_TIMEOUT` | None |
| `logger.Config` | `LOGGER_FORMAT`, `LOGGER_LEVEL` | None |
| `SQLiteConfig` | `SQLITE_PATH`, `SQLITE_AUTO_MIGRATE` | `loop.db`, `true` |
| `ProviderConfig` | `PROVIDER_BASE_URL`, `PROVIDER_API_KEY_OPENCODE`, `PROVIDER_MODEL`, `PROVIDER_REASONING_EFFORT` | None |
| `TelegramConfig` | `TELEGRAM_API_KEY`, `TELEGRAM_CHANNEL_ID` | None |
| `TavilyConfig` | `TAVILY_API_KEY` | None |
| `BrowserConfig` | `browser_ENABLED`, `browser_HEADLESS`, `browser_BIN`, `browser_CDP_URL`, `browser_TIMEOUT` | `true`, `true`, empty, empty, `60s` |

The browser prefix is lowercase in the struct tag, so its environment variable
names are the lowercase `browser_*` names shown above.

## Validation

- `SERVER_BIND_ADDR` and `SQLITE_PATH` must not be empty.
- Provider, Telegram, and Tavily values must not be empty.
- `browser_TIMEOUT` must be positive.
- Logger format must be `json` or `text`; logger level must be a valid slog level.

## Provider

The `/zen/go/v1` provider endpoint uses Chat Completions. The OpenCode Zen
`/zen/v1` endpoint uses the Responses API. Select the base URL and model
together. The free Zen model is `muse-spark-1.2-contributor-free`.

## Agent Tools

Telegram settings configure the `telegram_send` and `telegram_read` tools.
Tavily configures `websearch`. Browser tools are enabled by default and use a
local browser unless `browser_CDP_URL` points to an existing Chrome DevTools
Protocol endpoint; set `browser_ENABLED=false` to disable them.
