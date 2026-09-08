# `cmd/main`

The `taskman` CLI entrypoint - see `notes/design/design.md` §7. `main()` builds the `urfave/cli`
v3 command tree (`rootCommand`) and runs it once; there is no persistent daemon.

```
taskman [--git-dir <dir>] [--tasks-dir <dir>] [--worktrees-dir <dir>] [--json]
        [--log-level <level>] [--log-format <format>]
        task <command> [args...]
```

`task <command>` covers the full command table in design.md §6/§7: `list`, `get`, `create`,
`update`, `specify`, `implement`, `verify`, `review record`/`approve`/`reject`, `commit`,
`escalate`, `merge`, `abandon`, `next`, `delete`.

Every command's `Action` parses its own flags into the request struct `internal/task`'s matching
function expects, calls straight into that function, and renders the result: the full JSON
envelope behind `--json`, a short human-readable summary (via `internal/prompts`) otherwise.
`helpers.go` holds the shared plumbing (`repoFrom`/`gitFrom`, flag parsing, output rendering,
and the error-to-exit-code mapping in `fail`/`exitCode`).

Usage:

```sh
go run ./cmd/main task create --definition "Fix doc drift in balance package"
```

No environment variables, no config file - every directory taskman needs is a CLI flag with a
default (see design.md §1).
