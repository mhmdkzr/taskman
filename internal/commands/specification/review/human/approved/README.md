# specification review human approved

Records a human's approval of a task's specification.

Appends `SpecificationReviewHumanApproved` and routes to `implement`. Valid only in
`specification_review` and only when the specification's human review is required. Invoke it only
after a human explicitly supplies the decision.

## CLI

```bash
taskman specification review human approved --id <id> [--comment "..."]
```

## MCP

`task_specification_review_human_approved`.
