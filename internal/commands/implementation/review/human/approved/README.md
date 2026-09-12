# implementation review human approved

Records a human's approval of a task's implementation.

Appends `ImplementationReviewHumanApproved` and routes to `merge`. Valid only in `human_review` and
only when the implementation's human review is required. Invoke it only after a human explicitly
supplies the decision.

## CLI

```bash
taskman implementation review human approved --id <id> [--comment "..."]
```

## MCP

`task_implementation_review_human_approved`.
