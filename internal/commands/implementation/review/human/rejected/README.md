# implementation review human rejected

Records a human's rejection of a task's implementation, for another fix-and-verify round.

Appends `ImplementationReviewHumanRejected` with a required `--reason` and routes to
`fix_human_review_findings`. That state loops through `verify` to a new `commit` and back to
`human_review`; it does not repeat automated review. Valid only when the implementation's human
review is required.

## CLI

```bash
taskman implementation review human rejected --id <id> --reason "rate-limit headers are missing"
```

## MCP

`task_implementation_review_human_rejected`.
