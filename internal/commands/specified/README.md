# specified

Records a task's drafted specification and every requirement/policy decision for its eventual
implementation: which review gates apply (both the specification's own and the implementation's),
which verification checks the implementation will require, and whether it happens in a fresh
worktree (and if so, where).

`specified` builds a `SpecificationSubmitted` event and applies it via `store.Append` (validated by
`task.Apply` before persisting). From `specify` it routes to `specification_review` when either
specification-review gate is required, otherwise straight to `implement`.

`specified` may be called again whenever the task is back in `specify` after a rejection: each
submission replaces the specification wholesale and clears prior review results, so every policy
declared here can still be reconfigured until the task leaves `specify`.

Fixing implementation-time policy here - instead of at `implemented`, which used to accept it -
means `implemented` becomes pure fact-reporting: it copies this policy forward automatically
rather than asking for it again.

## Request

| Field | Flag | Required |
|---|---|---|
| `ID` | `--id` | yes |
| `Plan` | `--plan` | yes |
| `Review` | `--agent-review` / `--human-review` (+ auto-fix flags) | no |
| `Verification` | `--unit` / `--integration` / `--end-to-end` / `--linters` (+ auto-fix flags) | no |
| `ImplementationReview` | `--impl-agent-review` / `--impl-human-review` (+ auto-fix flags) | no |
| `Worktree` | `--use-worktree` (+ `--worktree` / `--branch`, required together with it) | no |

`ImplementationReview` requires at least one `Verification` check; the resulting task is rejected
otherwise (mirroring `Review`'s equivalent invariant, and matching the one `Implementation.validate`
already enforced when this policy used to live on `implemented`).

The review flags are shared between the two gates (and with `implemented` in the past) via
`utils.ReviewFlags(prefix)`; see `internal/utils`. `Review` uses the empty prefix, `ImplementationReview`
uses `impl-`.

## CLI

```bash
taskman specified --id <id> --plan "..." --agent-review \
  --unit --impl-human-review \
  --use-worktree --worktree .worktrees/<id> --branch feat/x
```

## MCP

`task_specified`.
