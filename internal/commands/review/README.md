# `internal/commands/review`

The review stage has three command slices: `reviewed` records the automated review
round's verdict, while `review approved` and `review rejected` are the human gate.
`cmd.go` assembles the human commands into the `review` command tree and exposes
the automated command separately at the root.

Each slice follows the same shape as the other commands: `cmd.go` and `mcp.go` are frontends,
while `<name>.go` validates its request, constructs a typed review event, and applies it through
the pure workflow inside `store.Update`. Automated approval advances to `commit`; rejection
routes to a fix or blocks after the configured second round. Human approval routes to `merge`
(or completes a trunk task), while rejection enters `fix_human_review_findings`.
