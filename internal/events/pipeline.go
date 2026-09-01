package events

import "time"

// PipelinePhase is published when the codebase/task/commit pipeline (see
// internal/pipeline) moves into a new phase for one task run. Phases are:
// clone, execution, review, commit, land, cleanup.
type PipelinePhase struct {
	TaskID    string    `json:"task_id"`
	Phase     string    `json:"phase"`
	Timestamp time.Time `json:"timestamp"`
}

func (PipelinePhase) Subject() string { return "agent.pipeline.phase" }

func (e PipelinePhase) MsgID() string { return eventID(e) }

// PipelineTaskFinished is published when a pipeline run completes
// successfully, with the commit hash landed in the source repository.
type PipelineTaskFinished struct {
	TaskID     string    `json:"task_id"`
	CommitHash string    `json:"commit_hash"`
	Timestamp  time.Time `json:"timestamp"`
}

func (PipelineTaskFinished) Subject() string { return "agent.pipeline.task.finished" }

func (e PipelineTaskFinished) MsgID() string { return eventID(e) }

// PipelineTaskFailed is published when a pipeline run returns an error,
// naming the phase it failed in.
type PipelineTaskFailed struct {
	TaskID    string    `json:"task_id"`
	Phase     string    `json:"phase"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

func (PipelineTaskFailed) Subject() string { return "agent.pipeline.task.failed" }

func (e PipelineTaskFailed) MsgID() string { return eventID(e) }

// DiffFile is one file's change in a task's unified diff, as shown to the
// agents (see internal/codebase.Diff).
type DiffFile struct {
	Name       string `json:"name"`
	ChangeType string `json:"change_type"`
	Additions  int    `json:"additions"`
	Deletions  int    `json:"deletions"`
	Patch      string `json:"patch"`
}

// PipelineDiff is published when the pipeline has the task's current unified
// diff, so a read-only UI can show it live. It is published at the start of
// review and whenever a fixup round changes the work.
type PipelineDiff struct {
	TaskID    string     `json:"task_id"`
	Files     []DiffFile `json:"files"`
	Timestamp time.Time  `json:"timestamp"`
}

func (PipelineDiff) Subject() string { return "agent.pipeline.diff" }

func (e PipelineDiff) MsgID() string { return eventID(e) }
