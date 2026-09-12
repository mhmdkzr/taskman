# `internal/task/store`

SQLite-backed, event-sourced persistence for `internal/task`. A task's current state is never
stored directly - it is always derived by replaying its events, in order, through `task.Apply`.

## Schema

Two tables (`store.go`): `tasks` (id, JSON definition, created_at) and `events` (task_id, seq,
kind, JSON data). The schema is created on `Open`.

## API

- `Open(path)` - opens (creating if needed) the SQLite database with WAL, a busy timeout, and
  foreign keys enabled, and ensures the schema exists. `Close` releases it.
- `Create(ctx, id, definition, at)` - inserts a new task seeded at `task.StateSpecify` by
  `task.NewTask`. Returns `ErrTaskAlreadyExists` on a duplicate id.
- `Read(ctx, id)` - loads the definition, then replays each event through `task.Apply`. Returns
  `ErrTaskNotFound` when the id is unknown.
- `Append(ctx, id, event)` - reads the current task, validates the event via `task.Apply`, and
  inserts it as the next `seq` - all inside one `BEGIN IMMEDIATE` transaction, so concurrent
  appends serialize and a rejected event is never written.
- `List(ctx)` - every task id, ordered by creation time.

Event encoding/decoding (`record.go`) is a `kind`-keyed switch over the concrete `task.TaskEvent`
types; an unknown kind surfaces as a corrupt-log error. `errors.go` holds the exported sentinels
(`ErrTaskNotFound`, `ErrTaskAlreadyExists`) and the internal corrupt-log error.
