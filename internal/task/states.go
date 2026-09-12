package task

import "fmt"

type TaskState string

const (
	StateSpecify                    TaskState = "specify"
	StateSpecificationReview        TaskState = "specification_review"
	StateImplement                  TaskState = "implement"
	StateVerify                     TaskState = "verify"
	StateFixVerificationFailure     TaskState = "fix_verification_failure"
	StateFixAutomatedReviewFindings TaskState = "fix_automated_review_findings"
	StateAutomatedReview            TaskState = "automated_review"
	StateCommit                     TaskState = "commit"
	StateHumanReview                TaskState = "human_review"
	StateFixHumanReviewFindings     TaskState = "fix_human_review_findings"
	StateMerge                      TaskState = "merge"
	StateBlocked                    TaskState = "blocked"
	StateCompleted                  TaskState = "completed"
	StateAbandoned                  TaskState = "abandoned"
)

func reviewRequired(review ReviewConfiguration) bool {
	return agentReviewRequired(review) || humanReviewRequired(review)
}

func agentReviewRequired(review ReviewConfiguration) bool {
	return review.Agent.Required
}

func humanReviewRequired(review ReviewConfiguration) bool {
	return review.Human.Required
}

// ParseTaskState validates and returns the TaskState corresponding to name.
func ParseTaskState(name string) (TaskState, error) {
	state := TaskState(name)
	if !state.valid() {
		return "", fmt.Errorf("unknown task state %q", name)
	}
	return state, nil
}

func (s TaskState) String() string {
	return string(s)
}

func (s TaskState) valid() bool {
	switch s {
	case StateSpecify, StateSpecificationReview, StateImplement, StateVerify,
		StateFixVerificationFailure, StateFixAutomatedReviewFindings,
		StateAutomatedReview, StateCommit, StateHumanReview,
		StateFixHumanReviewFindings, StateMerge, StateBlocked,
		StateCompleted, StateAbandoned:
		return true
	default:
		return false
	}
}
