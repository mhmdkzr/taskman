package list

import (
	"errors"
	"testing"

	taskrepo "github.com/mhmdkzr/loop/internal/agent/tools/task"
)

func TestInputValidateRejectsInvalidLevel(t *testing.T) {
	err := (Input{Filter: taskrepo.TaskFilter{Importance: []taskrepo.Level{taskrepo.Level(99)}}}).Validate()
	if !errors.Is(err, taskrepo.ErrInvalidLevel) {
		t.Fatalf("Validate() error = %v, want invalid level", err)
	}
}
