# agent sessions

Drives a stateless conversational agent loop over a session persisted in
SQLite.

## What it is

A session is an `agent_sessions` row plus an append-only `session_turns`
history: each turn is one user prompt paired with the full `goai.TextResult` it
produced. Every session runs as some agent — there is no session without one.
The sessions package exposes pure functions — `Create`, `Run` — that take the
current database handle and session ID on every call. There is no in-memory
mutable state to keep in sync; all state lives in the database.

## Behaviour

- `sessions.Create` resolves the named agent's model and prompt template,
  renders the system prompt with the given params, and creates an
  `agent_sessions` row with the given user, the agent's model, the rendered
  system prompt, provider options (including `reasoning_effort` from
  `config.ProviderConfig`), and an
  optional parent session ID (nil for a top-level, user-initiated session; set
  when this session is a dispatched subagent run). It returns the new session
  ID. The turn history starts empty.
- `sessions.Run` reloads the session's configuration and full turn history from
  the database, appends the new user prompt, calls the configured provider via
  `goai.GenerateText`, and appends the prompt + result as a new
  `session_turns` row. It returns the generated `*goai.TextResult`. The
  tool-call loop is bounded by a single global `maxSteps` constant, not a
  per-session or per-agent value.

Because `Run` is stateless, resuming a session is just calling `Run` again
with the same ID — the history is reconstructed from the database each time.

## Invocation

`cmd/loop` is a REPL around this package:

```sh
# new session (owner user must exist)
go run ./cmd/loop -user <user-uuid>

# resume an existing session
go run ./cmd/loop -resume <session-uuid>
```

Each stdin line is a prompt; the model's text response is printed to stdout.
