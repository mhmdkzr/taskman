# `internal/task/store`

SQLite-backed, event-sourced persistence for `internal/task`. A task's current state is never
stored directly - it is always derived by replaying its events, in order, through `task.Apply`.

## Schema

Two tables (`store.go`): `tasks` (id, JSON definition, created_at) and `events` (task_id, seq,
kind, JSON data). The schema is created on `Open`.

## API

- `Open(ctx, path)` - opens (creating if needed) the SQLite database with WAL, a busy timeout, and
  foreign keys enabled, and ensures the schema exists. `Close` releases it, logging a close
  failure at `Error` level before returning it (one-shot commands usually `defer` it, so the
  error must not be silently dropped).
- `Create(ctx, id, definition, at)` - inserts a new task seeded at `task.StateSpecify` by
  `task.NewTask`. Returns `ErrTaskAlreadyExists` on a duplicate id.
- `Read(ctx, id)` - loads the definition, then replays each event through `task.Apply`. Returns
  `ErrTaskNotFound` when the id is unknown.
- `Append(ctx, id, event)` - reads the current task, validates the event via `task.Apply`, and
  inserts it as the next `seq` - all inside one `BEGIN IMMEDIATE` transaction, so concurrent
  appends serialize and a rejected event is never written.
- `Delete(ctx, id)` - permanently removes the task and its entire event log in one transaction.
  The sole non-append-only operation, and the only one that never replays the log: it resolves the
  task by identity alone, so a task whose events can't be replayed is still removable. Fails with
  `ErrTaskNotFound` when the id is unknown.
- `List(ctx)` - every task id, ordered by creation time.

Event decoding (`record.go`) delegates to `task.DecodeEvent`, the one place mapping each
`EventKind` to its concrete `task.TaskEvent` type - this package keeps no `kind`-keyed switch of
its own to drift out of sync with it. `task.ErrUnknownEventKind` becomes this package's own
corrupt-log error; any other decode failure (a malformed payload for an otherwise-known kind)
bubbles up as-is. `errors.go` holds the exported sentinels (`ErrTaskNotFound`,
`ErrTaskAlreadyExists`) and the internal corrupt-log error.
