# implemented

Records that an implementation attempt is ready. All requirement/policy for this - which
verification checks it requires, which review gates apply, whether it happens in a fresh worktree
and where - was already decided at `specified` time (see `internal/commands/specified`); this
command is pure fact-reporting.

`implemented` reads the task's `Specification` and copies its `Verification`, `ImplementationReview`,
and `Worktree` policy forward into the recorded `Implementation` verbatim. If the specification
declared no fresh worktree (`Worktree.UseWorktree == false`), the current repository's own worktree
root and checked-out branch are auto-detected via `git rev-parse` instead of being asked for -
taskman still doesn't create a worktree or branch either way, it only reports where the
implementation happened.

A task specified before this policy moved to `specified` has none of it recorded; `implemented`
rejects such a task with an error asking for `specified` to be called again to declare it, rather
than guessing.

From `implement`, the task routes to `verify` if any check is required, else `automated_review` if
the agent review is required, else straight to `commit` - unchanged from before, since the routing
guards still read the same `Implementation.Verification`/`Implementation.Review` fields, just
populated from a different source.

## Request

| Field | Flag | Required |
|---|---|---|
| `ID` | `--id` | yes |

## CLI

```bash
taskman implemented --id <id>
```

## MCP

`task_implemented`.
