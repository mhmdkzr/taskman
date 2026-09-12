# specification review agent rejected

Records an agent reviewer's findings against a task's specification.

Appends `SpecificationReviewAgentRejected` and returns the task to `specify` for revision. Each
finding is a `location=detail` pair (`--finding`, repeatable); at least one is required, and valid
only when the specification's agent review is required.

## CLI

```bash
taskman specification review agent rejected --id <id> --finding plan=missing-rollback
```

## MCP

`task_specification_review_agent_rejected`.
