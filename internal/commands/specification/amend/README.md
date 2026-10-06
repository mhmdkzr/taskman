# specification amend

Revises a task's specification after it has been submitted but before its implementation is
recorded. It exists so a mis-specified task is never stuck: `specified` is only accepted in the
`specify` state, while `amend` is accepted from `specification_review` and `implement`.

Every field is a patch: only the flags (CLI) or JSON fields (MCP) the caller actually supplies are
applied over the task's current specification. An omitted field keeps its current value, so a
targeted change - enabling auto-fix, adding a verification check - never silently drops a gate.

The recorded specification is replaced wholesale, but its outcome is preserved selectively:

- If the amendment changes the specification review's **inputs** (the plan or either review gate's
  configuration), the review results are cleared and any still-required specification review runs
  again - even from `implement`, where the task returns to `specification_review`.
- Otherwise (only implementation policy changed) the recorded review result stands and the task
  stays where it is.

## CLI

```bash
# Turn on auto-fix for the specification's automated review, keeping every other field.
taskman specification amend --id <id> \
  --agent-review --agent-review-auto-fix \
  --agent-review-auto-fix-max-rounds 2 --agent-review-auto-fix-use-subagent

# Replace the plan.
taskman specification amend --id <id> --plan "revised plan"

# Add a verification check.
taskman specification amend --id <id> --integration
```

## MCP

`task_specification_amend`. The tool's input mirrors `task_specified`'s, with every field optional;
supply only the fields to change.
