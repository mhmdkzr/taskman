# `internal/task/view/json`

Projects a loaded task into the machine-readable `Document` that the CLI prints behind `--json`
and that every MCP tool returns:

```go
type Document struct {
	Task        task.Task
	State       task.TaskState
	Instruction task.Instruction
	Message     string
	Commands    []string
}
```

`FromTask(t)` derives `state`, `instruction`, and - via the
[`instructions`](../instructions) projection - the per-state `message` and the `commands` that
would currently report an outcome. This is presentation, never persistence - unlike the store's
own task serialization, it carries the derived fields an agent needs to drive the task.
