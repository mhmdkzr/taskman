# `internal/commands/review`

The review stage owns the human gate: `review approved` and `review rejected`.
The root-level `automated-review` slice records an automated review round's verdict.
`cmd.go` assembles the two human commands into the `review` command tree.

Each human-decision slice follows the same shape as the other commands: `cmd.go` and `mcp.go` are
frontends, while `<name>.go` validates its request, constructs a typed event, and applies it
through the pure workflow inside `store.Update`. Approval routes to `merge` (or completes a trunk
task), while rejection enters `fix_human_review_findings`.
