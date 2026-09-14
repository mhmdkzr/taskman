package task

import "fmt"

// markAutoFixBudgetExhausted records a blockage on t when config declares a
// finite auto-fix budget and rounds already spent exceed it. It is a no-op
// when auto-fix is disabled, the budget is unlimited (MaxRounds <= 0), or
// rounds remain. A set Blockage makes the owning transition route to
// StateBlocked instead of looping into another fix state, so a task whose
// automatic fixing has run out of rounds stops and waits for intervention.
func markAutoFixBudgetExhausted(t *Task, stage string, config AutoFix, rounds int) {
	if !config.Enabled || config.MaxRounds <= 0 || rounds <= config.MaxRounds {
		return
	}
	t.Blocked = &Blockage{
		Stage:  stage,
		Reason: fmt.Sprintf("auto-fix budget exhausted after %d round(s), max %d", rounds, config.MaxRounds),
	}
}

// failedVerificationCount is the number of verification attempts recorded so
// far that did not pass.
func failedVerificationCount(t *Task) int {
	count := 0
	for _, attempt := range t.Implementation.Verification.Attempts {
		if !attempt.Passed {
			count++
		}
	}
	return count
}

// rejectedAgentReviewCount is the number of automated implementation reviews
// recorded so far that did not approve.
func rejectedAgentReviewCount(t *Task) int {
	count := 0
	for _, result := range t.Implementation.Review.Agent.Results {
		if !result.Approved {
			count++
		}
	}
	return count
}

// rejectedHumanReviewCount is the number of human implementation reviews
// recorded so far that did not approve.
func rejectedHumanReviewCount(t *Task) int {
	count := 0
	for _, result := range t.Implementation.Review.Human.Results {
		if !result.Approved {
			count++
		}
	}
	return count
}
