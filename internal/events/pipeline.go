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
