# specification review agent approved

Records an independent agent reviewer's approval of a task's specification.

Appends `SpecificationReviewAgentApproved`. Valid only in `specification_review` and only when the
specification's agent review is required (guard `specAgentReviewRequired`), so a gate not
configured at `specified` is refused. It routes to `implement`, unless the human gate is also
required, in which case the task stays in `specification_review` for the human decision.

## CLI

```bash
taskman specification review agent approved --id <id> [--comment "..."]
```

## MCP

`task_specification_review_agent_approved`.
