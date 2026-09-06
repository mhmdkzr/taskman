package update

import (
	"testing"
	"uuid"

	taskrepo "github.com/mhmdkzr/loop/internal/agent/tools/task"
)

func validInput() Input {
	return Input{
		ID:            uuid.NewV7().String(),
		Definition:    "definition",
		Specification: "specification",
		State:         taskrepo.TaskStateCreated,
		Importance:    taskrepo.LevelLow,
		Urgency:       taskrepo.LevelLow,
		Complexity:    taskrepo.LevelLow,
		Effort:        taskrepo.LevelLow,
		Risk:          taskrepo.LevelLow,
		Autonomy:      taskrepo.LevelLow,
	}
}

func TestInputValidate(t *testing.T) {
	input := validInput()
	if err := input.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}

	input.State = "invalid"
	if err := input.Validate(); err == nil {
		t.Fatal("Validate() returned nil for invalid state")
	}
}
