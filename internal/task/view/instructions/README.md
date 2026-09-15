# `internal/task/view/instructions`

Projects a loaded task's current workflow position into actionable instructions - the
`Instructions` struct `json.Document` embeds into every command's output:

```go
type Instructions struct {
	TaskID   uuid.UUID
	State    task.TaskState
	Action   task.InstructionAction
	Message  string
	Commands []string
}
```

`Message` is rendered per state from the embedded `instructions.md` templates, against the
task's own data: its specification plan, worktree/branch, a failed check's output, a rejected
review's findings or comment, and the applicable auto-fix policy. `Commands` is derived from
`task.ValidEvents`, the same guards `task.Apply` enforces, so it never lists a command the store
would reject; distinct events reported through the same command (verification pass/fail) collapse
into one entry.

`Project(t)` is pure presentation, never persistence - it performs no transition and cannot
decide a review's verdict or a check's result.
