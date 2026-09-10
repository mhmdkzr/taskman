# update

`taskman update <id>` patches non-workflow metadata such as title, labels,
references, trunk mode, and auto-approval. It uses the value-oriented locked
task store but deliberately emits no workflow event and never changes the
current state. CLI and MCP call the same function.

Auto-approval is the one field with a state precondition: it is only read
when a commit is recorded, so changing it once a task's state is
`AutoApproveMoot` (`human_review`, `merge`, `blocked`, or either terminal
state) can no longer affect the task's outcome. `update` rejects such a
change with `task.ErrAutoApproveTooLate` instead of silently patching a flag
that no longer does anything.
