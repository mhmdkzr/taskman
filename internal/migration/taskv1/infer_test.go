package taskv1

import (
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
)

func TestInferState(t *testing.T) {
	t.Parallel()
	before := time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC)
	after := before.Add(time.Hour)
	done := legacyStageStatus{State: "done", CompletedAt: &before}
	passing := task.Verification{
		Checks:    map[string]task.CheckResult{"test": task.CheckOK},
		CreatedAt: after,
	}
	failing := task.Verification{
		Checks:    map[string]task.CheckResult{"test": task.CheckError},
		CreatedAt: after,
	}

	tests := []struct {
		name       string
		legacy     legacyTask
		want       task.State
		wantResume task.State
	}{
		{"completed", legacyTask{State: "completed"}, task.StateCompleted, ""},
		{"failed", legacyTask{State: "failed"}, task.StateAbandoned, ""},
		{"awaiting specification", legacyTask{State: "created"}, task.StateSpecify, ""},
		{
			"awaiting implementation",
			legacyTask{State: "started", Status: legacyStatus{Specification: done}},
			task.StateImplement,
			"",
		},
		{
			"awaiting first verification",
			legacyTask{
				State:  "started",
				Status: legacyStatus{Specification: done, Implementation: done},
			},
			task.StateVerify,
			"",
		},
		{
			"fixing failed verification",
			legacyTask{
				State:         "started",
				Status:        legacyStatus{Specification: done, Implementation: done},
				Verifications: []task.Verification{failing},
			},
			task.StateFixVerificationFailure,
			"",
		},
		{
			"awaiting automated review",
			legacyTask{
				State:         "started",
				Status:        legacyStatus{Specification: done, Implementation: done},
				Verifications: []task.Verification{passing},
			},
			task.StateAutomatedReview,
			"",
		},
		{
			"fixing rejected automated review",
			legacyTask{
				State:         "started",
				Status:        legacyStatus{Specification: done, Implementation: done},
				Verifications: []task.Verification{passing},
				Reviews:       []task.Review{{CreatedAt: after.Add(time.Minute)}},
			},
			task.StateFixAutomatedReviewFindings,
			"",
		},
		{
			"awaiting commit",
			legacyTask{
				State:  "started",
				Status: legacyStatus{Specification: done, Implementation: done, Verification: done},
			},
			task.StateCommit,
			"",
		},
		{
			"awaiting human review",
			legacyTask{
				State:  "started",
				Status: legacyStatus{Specification: done, Implementation: done, Verification: done},
				Git:    task.Git{Commit: &task.GitCommit{At: after}},
			},
			task.StateHumanReview,
			"",
		},
		{
			"fixing human rejection",
			legacyTask{
				State: "started",
				Status: legacyStatus{
					Specification:  done,
					Implementation: done,
					Verification:   done,
					Review:         legacyStageStatus{State: "in_progress"},
				},
				HumanReviews: []task.HumanReview{{At: before}},
			},
			task.StateFixHumanReviewFindings,
			"",
		},
		{
			"committing human-review fix",
			legacyTask{
				State: "started",
				Status: legacyStatus{
					Specification:  done,
					Implementation: done,
					Verification:   done,
					Review:         legacyStageStatus{State: "in_progress"},
				},
				HumanReviews:  []task.HumanReview{{At: before}},
				Verifications: []task.Verification{passing},
			},
			task.StateCommit,
			"",
		},
		{
			"awaiting merge",
			legacyTask{
				State: "started",
				Status: legacyStatus{
					Specification:  done,
					Implementation: done,
					Verification:   done,
					Review:         done,
				},
			},
			task.StateMerge,
			"",
		},
		{
			"blocked while specifying",
			legacyTask{State: "blocked"},
			task.StateBlocked,
			task.StateSpecify,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, resume, err := inferState(tt.legacy)
			if err != nil {
				t.Fatalf("inferState: %v", err)
			}
			if got != tt.want || resume != tt.wantResume {
				t.Fatalf(
					"inferState = (%q, %q), want (%q, %q)",
					got,
					resume,
					tt.want,
					tt.wantResume,
				)
			}
		})
	}
}
