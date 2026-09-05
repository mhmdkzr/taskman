# Agent Models

The `models` package refreshes the OpenCode model catalog from its
OpenAI-compatible `GET /models` endpoint and stores the returned models in
SQLite.

The endpoint does not expose thinking capabilities. Refresh therefore stores
`low`, `medium`, and `high` `reasoning_effort` options by default, with
currently known OpenCode exceptions for DeepSeek V4, GLM 5.2, and MiniMax M3.

## CLI

```sh
# Refresh the local catalog from OpenCode.
go run ./cmd/main models -refresh

# Print the stored catalog as JSON.
go run ./cmd/main models -list -json

# Refresh only the selected provider.
go run ./cmd/main models -refresh -provider opencode
```

The web UI port can be overridden for the normal server command:

```sh
go run ./cmd/main -port 8081

# Bind to a specific address and port.
go run ./cmd/main -addr 127.0.0.1 -port 8081

# Use a specific SQLite database file.
go run ./cmd/main -db ./data/loop.sqlite

# Override the env file and logger settings for one run.
go run ./cmd/main -env .env.local -log-level debug -log-format json
```
