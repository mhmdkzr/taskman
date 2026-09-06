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
  `session_turns` row. It returns the generated `*goai.TextResult`.

Because `Run` is stateless, resuming a session is just calling `Run` again
with the same ID — the history is reconstructed from the database each time.

## Crash recovery

A `session_turns` row has a `status`: `running`, `completed`, or
`interrupted`. `Run` inserts a row as `running` (the write-ahead marker)
*before* calling `goai.GenerateText`, so the turn exists durably even if the
process dies mid-loop — not just after the whole loop returns. While the loop
runs, `goai` hooks (`OnStepFinish`, `OnToolCallStart`, `OnToolCall`) write each
step and tool call to `session_turn_events` as it happens, so a crash mid-turn
leaves a real trace of what was attempted, not silence. Once
`GenerateText` returns normally, the row is updated to `completed` with the
full result.

A row still `running` was orphaned by a crash (or a same-process error, which
`Run` closes out immediately rather than leaving `running`). `sessions.
ReconcileInterrupted` — called once at process boot, before any session is
resumed — finds every such row and replays its `session_turn_events` into a
synthesized partial result: each tool call the model requested gets a real
result (if one was recorded) or a synthesized error distinguishing "never
attempted" from "started but its outcome is unknown, verify before retrying"
— never a dangling tool call with no result, which would be an invalid
message sequence for the next model call. The row is then marked
`interrupted`. Recovery makes the stored conversation valid to resume; it does
not undo or verify any side effects a tool call already caused before the
crash.

## Invocation

`cmd/loop` is a REPL around this package:

```sh
# new session (owner user must exist)
go run ./cmd/loop -user <user-uuid>

# resume an existing session
go run ./cmd/loop -resume <session-uuid>
```

Each stdin line is a prompt; the model's text response is printed to stdout.
