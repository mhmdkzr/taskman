# `internal/task/view/json`

Projects a loaded task into the machine-readable `Document` that the CLI prints behind `--json`
and that every MCP tool returns:

```go
type Document struct {
	Task        task.Task
	State       task.TaskState
	Instruction task.Instruction
}
```

`FromTask(t)` derives `state` and `instruction` from the task. This is presentation, never
persistence - unlike the store's own task serialization, it carries the two derived fields an
agent needs to drive the task.
