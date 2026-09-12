# implementation review agent rejected

Records an agent reviewer's findings against a task's implementation.

Appends `ImplementationReviewAgentRejected` with one or more `location=detail` findings and routes
to `fix_automated_review_findings`, which loops back through `verify` to `automated_review`. There
is no built-in round limit. Valid only when the implementation's agent review is required.

## CLI

```bash
taskman implementation review agent rejected --id <id> --finding handler.go=missing-timeout
```

## MCP

`task_implementation_review_agent_rejected`.
