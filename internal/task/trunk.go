package task

// CompleteTrunkMerge marks a trunk task's merge stage done and the task
// completed, the moment its review stage completes. A task created with
// --trunk has no real merge to record (design.md §5) - there's nothing left
// for a caller to do once review is done, so the merge stage completes
// automatically instead of waiting for an explicit merge call.
func CompleteTrunkMerge(t *Task) {
	if !t.Git.Trunk {
		return
	}
	t.Status.Merge = StageStatus{State: StageDone, CompletedAt: new(Now())}
	t.State = StateCompleted
}
