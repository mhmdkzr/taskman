# `internal/commands/review`

The review stage's three commands, one vertical slice package each: `recorded` (the automated
review round's verdict), `approved` and `rejected` (the human gate). `cmd.go` assembles them into
the `review` command tree, mounted on the root by `internal/commands`.

Each slice follows the same shape as the other commands: `cmd.go` and `mcp.go` are frontends,
while `<name>.go` validates its request, constructs a typed review event, and applies it through
the pure workflow inside `store.Update`. Automated approval advances to `commit`; rejection
routes to a fix or blocks after the configured second round. Human approval routes to `merge`
(or completes a trunk task), while rejection enters `fix_human_review_findings`.
