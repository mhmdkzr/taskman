# `internal/cli/task/review`

The review stage's three commands, one vertical slice package each: `record` (the automated
review round's verdict), `approve` and `reject` (the human gate). `review.go` assembles them into
the `review` command tree, nested under `task` by `internal/cli/task`.

Each slice follows the same shape as `internal/cli/task`'s other slices: `cmd.go` exports a
`Command()` func returning a `*cli.Command`, and `<name>.go` holds the domain logic - parse
flags, mutate the task via `internal/task`'s shared primitives, render via `internal/cli/support`.
None of these slices import `internal/cli/task` (that would cycle back through it) - anywhere
their tests need a task in a later stage, they drive `internal/task` directly rather than going
through the CLI commands.
