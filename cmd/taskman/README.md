# `cmd/taskman`

The human-facing CLI for the SQLite task backlog (`internal/task`) and the codebase → task → commit pipeline (`internal/pipeline`).

## What it provides

- `taskman add [flags]` — file a new task. `-title`, `-type`, `-urgency`, `-importance`, `-risk`, `-what`, `-why`, `-how` and at least one `-completed-when` are required; `-package`, `-where`, `-invariant`, `-tag`, `-depends-on` are repeatable and optional. Prints the new task's id.
- `taskman list [-status s]` — list tasks, optionally filtered by status (`created`/`started`/`completed`/`reviewed`).
- `taskman show <id>` — print one task in full, as JSON.
- `taskman run <id> [<id> ...] [-source dir] [-max-fixup-rounds n]` — publish a run command to the server (`internal/runner`), which drives the listed tasks through `pipeline.RunTask`, each in its own goroutine and worktree. Requires a reachable NATS bus (`NATS_URL`) with the server running; provider credentials are the server's, not the CLI's.
- `taskman reset <id>` — return a task stuck mid-lifecycle (e.g. a `run` that crashed between phases) to `created` so it can be re-run. Clears both session ids, commit info, token usage, and the started/completed/reviewed timestamps; fails if the task is already `created` or doesn't exist.

## How it's invoked

```sh
go run ./cmd/taskman add -title "Fix nil pointer" -type bug -urgency low -importance medium -risk low \
  -what "..." -why "..." -how "..." -completed-when "tests pass"

go run ./cmd/taskman list
go run ./cmd/taskman show <id>
go run ./cmd/taskman run <id> -source /path/to/repo
go run ./cmd/taskman run <id1> <id2> <id3> -source /path/to/repo
go run ./cmd/taskman reset <id>
```

Configuration is read from the same `AGENT_*` environment variables as the server (see `.env.example`); `AGENT_DB_PATH` (default `~/.taskman/taskman.db`) is the task store every subcommand operates on.
