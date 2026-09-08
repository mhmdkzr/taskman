# `internal/cli/task`

Every `taskman task <command>` subcommand, one Go file per command: `list.go`, `get.go`,
`create.go`, `update.go`, `specify.go`, `implement.go`, `verify.go`, `commit.go`, `escalate.go`,
`merge.go`, `abandon.go`, `next.go`, `delete.go`. `task.go` assembles them (plus the nested
`review` subcommand from `internal/cli/task/review`) into the `task` command tree.

Each file exports one function returning a `*cli.Command` - e.g. `Create()`, `Verify()`. The
`Action` parses flags into the request struct `internal/task`'s matching function expects, calls
it, and renders the result via `internal/cli/support` (`support.PrintTask`/`support.Fail`/etc.).
`internal/cli/support` exists specifically so this package and `review` can share that plumbing
without an import cycle back to `internal/cli`, which imports this package.

Every flag has a `Usage` string written for someone who only has the compiled binary - no
references to files or paths in this repo.
