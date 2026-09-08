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
| `GitHubConfig` | `GITHUB_TOKEN`, `GITHUB_REPO_URL`, `GITHUB_ISSUE_LABEL`, `GITHUB_POLL_INTERVAL`, `GITHUB_WEBHOOK_SECRET` | None, None, `loop-pipeline`, `1m`, None |

The browser prefix is lowercase in the struct tag, so its environment variable
names are the lowercase `browser_*` names shown above.

## Validation

- `SERVER_BIND_ADDR` and `SQLITE_PATH` must not be empty.
- Provider, Telegram, and Tavily values must not be empty.
- `GITHUB_TOKEN` and `GITHUB_REPO_URL` must not be empty; `GITHUB_WEBHOOK_SECRET`
  may be empty (its absence disables the webhook endpoint, see
  `internal/webhooks`). `GITHUB_POLL_INTERVAL` must be positive.
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

`GitHubConfig` drives the autonomous pipeline (see `internal/pipeline`),
which runs inside the main server process - no separate command.
`GITHUB_TOKEN` needs a personal access token with issues (read) and pull
requests (read/write) permission on the target repo. `GITHUB_REPO_URL` is
the repo it watches (a plain clone URL - `https://github.com/owner/repo.git`
or similar). `GITHUB_ISSUE_LABEL` (default `loop-pipeline`) is the opt-in
marker an issue needs to be picked up. `GITHUB_POLL_INTERVAL` (default `1m`)
is how often it re-scans for labeled issues as a baseline.
`GITHUB_WEBHOOK_SECRET`, if set, additionally enables `POST
/webhooks/github` to trigger a re-scan immediately on a GitHub `issues`
webhook delivery - see `internal/webhooks/README.md` for how to set that up.
