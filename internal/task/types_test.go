package task

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"uuid"
)

func TestTaskValidateRejectsStateWithoutRequiredData(t *testing.T) {
	input := fmt.Sprintf(`task:
  id: %s
  definition:
    description: do the work
  state-history:
    - state: merge
      at: 2024-01-01T00:00:00Z
`, uuid.NewV7())
	var decoded TaskDocument
	if err := yaml.Unmarshal([]byte(input), &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	if err := decoded.Task.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want an error")
	}
}

func TestReviewConfigurationValidateRejectsConfigWhenNotRequired(t *testing.T) {
	review := ReviewConfiguration{
		Agent: AgentReviewConfiguration{Results: []AgentReviewResult{{Approved: true}}},
	}
	if err := review.validate(); err == nil {
		t.Fatal("validate() error = nil, want an error for results without required")
	}
}

func TestHumanReviewConfigurationValidateRejectsConfigWhenNotRequired(t *testing.T) {
	review := HumanReviewConfiguration{Results: []HumanReviewResult{{Approved: true}}}
	if err := review.validate(); err == nil {
		t.Fatal("validate() error = nil, want an error for results without required")
	}
}

func TestImplementationValidateRejectsReviewWithoutVerification(t *testing.T) {
	impl := Implementation{Review: ReviewConfiguration{Agent: AgentReviewConfiguration{Required: true}}}
	if err := impl.validate(); err == nil {
		t.Fatal("validate() error = nil, want an error for a review gate without verification")
	}
}

func TestImplementationValidateAcceptsReviewWithVerification(t *testing.T) {
	impl := Implementation{
		Verification: Verification{Tests: TestConfiguration{Unit: true}},
		Review: ReviewConfiguration{
			Agent: AgentReviewConfiguration{Required: true},
			Human: HumanReviewConfiguration{Required: true},
		},
	}
	if err := impl.validate(); err != nil {
		t.Fatalf("validate() error = %v, want nil with verification configured", err)
	}
}

func TestVerificationValidateRejectsConfigWhenNotRequired(t *testing.T) {
	v := Verification{AutoFix: AutoFix{Enabled: true}}
	if err := v.validate(); err == nil {
		t.Fatal("validate() error = nil, want an error for auto-fix without any check required")
	}
}

func TestVerificationValidateRejectsUnrequiredCheckInAttempt(t *testing.T) {
	v := Verification{
		Tests:    TestConfiguration{Unit: true},
		Attempts: []VerificationResult{{Checks: Checks{Unit: CheckOK, Integration: CheckOK}}},
	}
	if err := v.validate(); err == nil {
		t.Fatal("validate() error = nil, want an error for a check that was never required")
	}
}

func TestVerificationValidateRejectsMissingRequiredCheckInAttempt(t *testing.T) {
	v := Verification{
		Tests:    TestConfiguration{Unit: true, Integration: true},
		Attempts: []VerificationResult{{Checks: Checks{Unit: CheckOK}}},
	}
	if err := v.validate(); err == nil {
		t.Fatal("validate() error = nil, want an error for a required check missing from an attempt")
	}
}

func TestValidateRejectsBlockedDataOutsideBlockedState(t *testing.T) {
	tsk := Task{
		ID:           uuid.NewV7(),
		Definition:   TaskDefinition{Description: "d"},
		StateHistory: []StateChange{{State: StateSpecify}},
		Blocked:      &Blockage{Stage: "s", Reason: "r"},
	}
	if err := tsk.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want an error for blocked data outside blocked state")
	}
}

func TestValidateRequiresBlockedDataInBlockedState(t *testing.T) {
	tsk := Task{
		ID:           uuid.NewV7(),
		Definition:   TaskDefinition{Description: "d"},
		StateHistory: []StateChange{{State: StateBlocked}},
	}
	if err := tsk.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want an error for missing blocked data")
	}
}

func TestValidateRejectsAbandonedDataOutsideAbandonedState(t *testing.T) {
	tsk := Task{
		ID:           uuid.NewV7(),
		Definition:   TaskDefinition{Description: "d"},
		StateHistory: []StateChange{{State: StateSpecify}},
		Abandoned:    &Abandonment{Reason: "r"},
	}
	if err := tsk.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want an error for abandoned data outside abandoned state")
	}
}

func TestValidateRequiresAbandonedDataInAbandonedState(t *testing.T) {
	tsk := Task{
		ID:           uuid.NewV7(),
		Definition:   TaskDefinition{Description: "d"},
		StateHistory: []StateChange{{State: StateAbandoned}},
	}
	if err := tsk.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want an error for missing abandoned data")
	}
}

func TestValidateRejectsEmptyStateHistory(t *testing.T) {
	tsk := Task{ID: uuid.NewV7(), Definition: TaskDefinition{Description: "d"}}
	if err := tsk.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want an error for empty state history")
	}
}

func TestNewTaskRejectsMissingID(t *testing.T) {
	if _, err := NewTask(uuid.Nil(), TaskDefinition{Description: "d"}, time.Now()); err == nil {
		t.Fatal("NewTask() error = nil, want an error for a missing id")
	}
}

func TestNewTaskRejectsMissingDescription(t *testing.T) {
	if _, err := NewTask(uuid.NewV7(), TaskDefinition{}, time.Now()); err == nil {
		t.Fatal("NewTask() error = nil, want an error for a missing description")
	}
}

func TestTaskYAMLRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tsk, err := NewTask(uuid.NewV7(), TaskDefinition{Description: "do the work"}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}

	encoded, err := yaml.Marshal(TaskDocument{Task: tsk})
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}
	if !strings.Contains(string(encoded), "state: specify") {
		t.Fatalf("encoded task does not contain state: %s", encoded)
	}

	var decoded TaskDocument
	if err := yaml.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	if decoded.Task.ID != tsk.ID || decoded.Task.State() != tsk.State() {
		t.Fatalf("decoded task = (%q, %q), want (%q, %q)",
			decoded.Task.ID, decoded.Task.State(), tsk.ID, tsk.State())
	}
}

func TestTaskJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tsk, err := NewTask(uuid.NewV7(), TaskDefinition{Description: "do the work"}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}

	encoded, err := json.Marshal(tsk)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(encoded), `"specify"`) {
		t.Fatalf("encoded task does not contain state: %s", encoded)
	}

	var decoded Task
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.ID != tsk.ID || decoded.State() != tsk.State() {
		t.Fatalf("decoded task = (%q, %q), want (%q, %q)",
			decoded.ID, decoded.State(), tsk.ID, tsk.State())
	}
}

func TestTaskStateUnmarshalTextRejectsUnknownState(t *testing.T) {
	var s TaskState
	if err := s.UnmarshalText([]byte("not-a-real-state")); err == nil {
		t.Fatal("UnmarshalText() error = nil, want an error")
	}
}
