# implementation review agent approved

Records an independent agent reviewer's approval of a task's implementation.

Appends `ImplementationReviewAgentApproved`. Valid only in `automated_review` and only when the
implementation's agent review is required (guard `implAgentReviewRequired`), so a gate not
configured at `implemented` is refused. It routes to `commit`.

## CLI

```bash
taskman implementation review agent approved --id <id> [--comment "..."]
```

## MCP

`task_implementation_review_agent_approved`.
