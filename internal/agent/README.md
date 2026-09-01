# `agent`

Library API for running an LLM agent with tools, wrapping `github.com/zendev-sh/goai` behind agent-owned types.

## What it provides

- `Run` / `Stream` — execute one prompt non-streaming or streaming, publishing events to the bus.
- `Session` — a multi-turn conversation with message history.
- `Consumer` — runs scheduled `agent.run` messages as a durable JetStream consumer.
- `PersistRun` / `ContinueSession` / `ForkRun` — persist runs into the shared SQLite store.
- `DefaultTools` — the tool set for one Options.Codebase (read/edit/glob/grep, telegram, spawn) and sub-agent `Runner`.

## How it's invoked

The server package builds a `Server` around `Options` (see `internal/server`); the application wires it into `internal/process/start.go`. Options carry the `config.AgentConfig` loaded from `AGENT_*` environment variables.

## Notes

- `Options.model` is unexported; only tests set it via `SetModelForTesting` with a `provider.LanguageModel`.
- The `agent.run` wire contract (`RunRequest`, `RunSubject`, `MeaningfulOverride`) lives in `internal/tools/schedule` and is re-exported here.