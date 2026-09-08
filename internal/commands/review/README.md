# `internal/commands/review`

The review stage's three commands, one vertical slice package each: `record` (the automated
review round's verdict), `approve` and `reject` (the human gate). `cmd.go` assembles them into
the `review` command tree, mounted on the root by `internal/commands`.

Each slice follows the same shape as `internal/commands`' other slices: `cmd.go` exports a
`Command()` func returning a `*cli.Command`, and `<name>.go` holds the domain logic - parse
flags, mutate the task via `internal/task`'s shared primitives, render via `internal/utils`.
None of these slices import `internal/commands` (that would cycle back through it) - anywhere
their tests need a task in a later stage, they drive `internal/task` directly rather than going
through the CLI commands.
