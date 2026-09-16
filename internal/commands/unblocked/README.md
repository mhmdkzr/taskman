# unblocked

Resumes a blocked task.

`unblocked` only applies from the `blocked` state. It resumes the task at whichever state it
occupied immediately before it became blocked - derived from the task's own `StateHistory`, not a
destination the caller picks.

A task blocks two ways: an `escalated` report (any stage/reason), or a fix loop running out of its
`--auto-fix-max-rounds` budget (stage `verification`, `automated review`, or `human review`). For
the latter, `--rounds` grants that many additional auto-fix rounds to the exhausted gate and is
required (must be `> 0`) - resuming without raising the budget would just land the task back in
the same fix state and immediately re-block on the next failure. For an escalation-caused
blockage there is no budget to grant, so `--rounds` must be omitted.

A budget grant is additive, recorded alongside the gate's other history - it never rewrites the
gate's original `--auto-fix-max-rounds`. The gate's effective budget is always its original max
plus the sum of every grant made against it.

## CLI

```bash
# resume an escalation
taskman unblocked --id <id> --reason "requirements clarified"

# resume a budget-exhaustion blockage, granting 2 more rounds
taskman unblocked --id <id> --reason "flaky test fixed upstream" --rounds 2
```

## MCP

`task_unblocked`.
