# `internal/cli/task/review`

The review stage's three commands, one file each: `record.go` (the automated review round's
verdict), `approve.go` and `reject.go` (the human gate). `review.go` assembles them into the
`review` command tree, nested under `task` by `internal/cli/task`.

Each file exports one function returning a `*cli.Command`, following the same shape as
`internal/cli/task`'s commands: parse flags, call into `internal/task`, render via
`internal/cli/support`. This package never imports `internal/cli/task` (that would cycle back
through it) - anywhere its own tests need a task in a later stage, they drive `internal/task`
directly rather than going through the CLI commands.
