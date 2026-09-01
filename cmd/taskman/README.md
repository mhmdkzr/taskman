# `cmd/taskman`

The human-facing CLI for the SQLite task backlog (`internal/task`) and the codebase → task → commit pipeline (`internal/pipeline`).

## What it provides

- `taskman add [flags]` — file a new task. `-title`, `-type`, `-urgency`, `-importance`, `-risk`, `-what`, `-why`, `-how` and at least one `-completed-when` are required; `-package`, `-where`, `-invariant`, `-tag`, `-depends-on` are repeatable and optional. Prints the new task's id.
- `taskman list [-status s]` — list tasks, optionally filtered by status (`created`/`started`/`completed`/`reviewed`).
- `taskman show <id>` — print one task in full, as JSON.
- `taskman run <id> [-source dir] [-workdir dir] [-max-fixup-rounds n]` — run `pipeline.RunTask` for one task: clones `-source` (default `.`) into an isolated worktree under `-workdir` (default a fresh temp dir), runs the execution/review/commit agents, and prints the resulting commit hash and session ids. Requires `AGENT_PROVIDER_BASE_URL` and `AGENT_PROVIDER_API_KEY`; `add`/`list`/`show` don't.

## How it's invoked

```sh
go run ./cmd/taskman add -title "Fix nil pointer" -type bug -urgency low -importance medium -risk low \
  -what "..." -why "..." -how "..." -completed-when "tests pass"

go run ./cmd/taskman list
go run ./cmd/taskman show <id>
go run ./cmd/taskman run <id> -source /path/to/repo
```

Configuration is read from the same `AGENT_*` environment variables as the server (see `.env.example`); `AGENT_DB_PATH` (default `~/.taskman/taskman.db`) is the task store every subcommand operates on.
