package task

type InstructionAction string

const (
	InstructionDispatch InstructionAction = "dispatch"
	InstructionRun      InstructionAction = "run"
	InstructionWait     InstructionAction = "wait"
	InstructionDone     InstructionAction = "done"
)

type InstructionKind string

const (
	InstructionSpecify                    InstructionKind = "specify"
	InstructionImplement                  InstructionKind = "implement"
	InstructionVerify                     InstructionKind = "verify"
	InstructionFixVerificationFailure     InstructionKind = "fix_verification_failure"
	InstructionFixAutomatedReviewFindings InstructionKind = "fix_automated_review_findings"
	InstructionAutomatedReview            InstructionKind = "automated_review"
	InstructionCommit                     InstructionKind = "commit"
	InstructionHumanReview                InstructionKind = "human_review"
	InstructionFixHumanReviewFindings     InstructionKind = "fix_human_review_findings"
	InstructionMerge                      InstructionKind = "merge"
	InstructionBlocked                    InstructionKind = "blocked"
	InstructionCompleted                  InstructionKind = "completed"
	InstructionAbandoned                  InstructionKind = "abandoned"
)

// Instruction is the pure workflow projection consumed by task next.
type Instruction struct {
	Kind   InstructionKind
	Action InstructionAction
}
