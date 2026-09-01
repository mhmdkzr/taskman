package events

// Command subjects carry requests into the system — the read-only UI's one
// way to make something happen. Commands are published over core NATS (see
// publisher.PublishCore), not the TASKMAN stream: they are transient
// requests, not durable events, so they carry no dedup MsgID.
const (
	// CommandRunSubject is where RunCommand requests are published.
	CommandRunSubject = "taskman.command.run"
)

// RunCommand asks the runner to drive one or more tasks through the pipeline
// (see internal/pipeline.RunTask). TaskIDs are the task ids to run.
type RunCommand struct {
	TaskIDs []string `json:"task_ids"`
}
