# next

Reports what should happen next for a task: a rendered guidance message plus the command(s) that
would currently report an outcome.

`next` performs no transition. It reads the task, projects its `task.Instruction` (state and
action), and renders a per-state message from the task's own data - the specification's plan, the
implementation's worktree/branch, a verification failure's checks and output, a rejected review's
findings or comment. The command list is derived from `task.ValidEvents`, the same guards `Apply`
enforces, so it never lists a command the store would actually reject; a state with more than one
currently-valid gate (e.g. `specification_review` when both agent and human review are required)
lists every one of them.

`next` cannot decide a review's verdict, a verification's actual check results, or draft a
specification/implementation - those require judgment or content only a human or an agent can
supply. For those states it explains what's expected and which command reports the outcome, but
never executes anything itself.

In a fix state (`fix_verification_failure`, `fix_automated_review_findings`,
`fix_human_review_findings`) the message also describes the gate's auto-fix policy - whether it is
enabled, its round cap, and whether the fix should run in a subagent - so the caller knows whether
to fix automatically and how many rounds remain.

## CLI

```bash
taskman next --id <id>
```

`--id` is required. Prints the message and command list, or the JSON envelope with `--json`.

## MCP

`task_next`.
