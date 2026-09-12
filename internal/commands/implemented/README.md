# implemented

Records that an implementation attempt is ready: the worktree and branch it was done in, which
verification checks it requires, and which review gates are required.

Taskman does not create the worktree or branch - the caller reports ones already created. From
`implement`, the task routes to `verify` if any check is required, else `automated_review` if the
agent review is required, else straight to `commit`.

## Request

| Field | Flag | Required |
|---|---|---|
| `ID` | `--id` | yes |
| `Worktree` | `--worktree` | yes |
| `Branch` | `--branch` | yes |
| `Verification` | `--unit` / `--integration` / `--end-to-end` / `--linters` (+ auto-fix flags) | no |
| `Review` | `--agent-review` / `--human-review` (+ auto-fix flags) | no |

A review gate requires at least one verification check; the resulting task is rejected otherwise.
Because an implementation is recorded only once per task, its review gates are fixed by this call
(unlike `specified`, which can be resubmitted).

## CLI

```bash
taskman implemented --id <id> --worktree /path --branch feat/x --unit --human-review
```

## MCP

`task_implemented`.
