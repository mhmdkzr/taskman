package task

import (
	"testing"
	"time"
	"uuid"
)

func TestCloneIsIndependentOfOriginal(t *testing.T) {
	now := time.Now().UTC()
	original := Task{
		ID:           uuid.NewV7(),
		Definition:   TaskDefinition{Description: "d", Labels: map[string]string{"k": "v"}},
		StateHistory: []StateChange{{State: StateSpecify, At: now}},
		Specification: &Specification{
			Plan: "p",
			Review: ReviewConfiguration{
				Agent: AgentReviewConfiguration{
					Required: true,
					Results:  []AgentReviewResult{{Findings: []Finding{{Location: "a"}}}},
				},
			},
		},
		Implementation: &Implementation{
			Git:          Git{Commits: []GitCommit{{Hash: "c1"}}, Merge: &GitMerge{Target: "main"}},
			Verification: Verification{Attempts: []VerificationResult{{Checks: Checks{Unit: CheckOK}}}},
		},
		Blocked:   &Blockage{Stage: "s", Reason: "r"},
		Abandoned: &Abandonment{Reason: "r"},
	}

	clone := original.Clone()

	clone.Definition.Labels["k"] = "changed"
	clone.StateHistory[0].State = StateCompleted
	clone.Specification.Plan = "changed"
	clone.Specification.Review.Agent.Results[0].Findings[0].Location = "changed"
	clone.Implementation.Git.Commits[0].Hash = "changed"
	clone.Implementation.Git.Merge.Target = "changed"
	clone.Implementation.Verification.Attempts[0].Checks.Unit = "changed"
	clone.Blocked.Reason = "changed"
	clone.Abandoned.Reason = "changed"

	switch {
	case original.Definition.Labels["k"] != "v":
		t.Error("mutating clone's Labels affected original")
	case original.StateHistory[0].State != StateSpecify:
		t.Error("mutating clone's StateHistory affected original")
	case original.Specification.Plan != "p":
		t.Error("mutating clone's Specification affected original")
	case original.Specification.Review.Agent.Results[0].Findings[0].Location != "a":
		t.Error("mutating clone's Findings affected original")
	case original.Implementation.Git.Commits[0].Hash != "c1":
		t.Error("mutating clone's Commits affected original")
	case original.Implementation.Git.Merge.Target != "main":
		t.Error("mutating clone's Merge affected original")
	case original.Implementation.Verification.Attempts[0].Checks.Unit != CheckOK:
		t.Error("mutating clone's Attempts affected original")
	case original.Blocked.Reason != "r":
		t.Error("mutating clone's Blocked affected original")
	case original.Abandoned.Reason != "r":
		t.Error("mutating clone's Abandoned affected original")
	}
}

func TestCloneHandlesNilOptionalFields(t *testing.T) {
	original := Task{ID: uuid.NewV7(), Definition: TaskDefinition{Description: "d"}}
	clone := original.Clone()
	if clone.Specification != nil || clone.Implementation != nil || clone.Blocked != nil || clone.Abandoned != nil {
		t.Fatal("Clone() populated fields that were nil on the original")
	}
}
