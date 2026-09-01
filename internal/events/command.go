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
// (see internal/pipeline.RunTask). TaskIDs are the task ids to run; Source is
// the git repository to clone each task's worktree from (default "." when
// empty). MaxFixupRounds bounds fixup rounds per phase; non-positive uses the
// pipeline's own default.
type RunCommand struct {
	TaskIDs        []string `json:"task_ids"`
	Source         string   `json:"source,omitempty"`
	MaxFixupRounds int      `json:"max_fixup_rounds,omitempty"`
}
