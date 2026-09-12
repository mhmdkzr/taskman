# specification review human rejected

Records a human's rejection of a task's specification, for revision.

Appends `SpecificationReviewHumanRejected` with a required `--reason` and returns the task to
`specify`. Valid only when the specification's human review is required, and only after a human
explicitly supplies the decision.

## CLI

```bash
taskman specification review human rejected --id <id> --reason "clarify error behavior"
```

## MCP

`task_specification_review_human_rejected`.
