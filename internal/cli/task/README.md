# `internal/cli/task`

Every `taskman task <command>` subcommand is its own vertical slice, one package per command:
`list`, `get`, `create`, `update`, `specify`, `implement`, `verify`, `commit`, `escalate`,
`merge`, `abandon`, `next`, `delete` (plus the nested `review` subcommand, itself split into
`review/record`, `review/approve`, `review/reject`). `task.go` assembles all of them into the
`task` command tree.

Each slice package holds `cmd.go` (the `Command()` func returning a `*cli.Command`) and a
`<name>.go` file with that command's domain logic - a plain function taking `tasksDir` (and any
other runtime deps) directly, built on `internal/task`'s shared primitives (`task.ReadTask`,
`task.WriteTaskFile`, `task.MutateTask`). Slices whose command dispatches an agent (`create`,
`specify`, `implement`) also own a `prompt.go` embedding that command's own `prompt.md` template.

`cmd.go`'s `Action` parses flags into the request the slice's domain function expects, calls it,
and renders the result via `internal/cli/support` (`support.PrintTask`/`support.Fail`/etc.).
`internal/cli/support` exists specifically so every slice here can share that plumbing without an
import cycle back to `internal/cli`, which imports this package. `next` is the one slice that
legitimately imports other slices (`specify`, `implement`), since guiding a task means knowing
what every stage's own commands would accept next, and none of those import `next` back.

Every flag has a `Usage` string written for someone who only has the compiled binary - no
references to files or paths in this repo.
