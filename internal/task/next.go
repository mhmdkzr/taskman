package task

import "fmt"

// Next returns the instruction attached to the task's current workflow state.
func Next(current Task) (Instruction, error) {
	state, ok := workflow.states[current.State]
	if !ok {
		return Instruction{}, fmt.Errorf("next instruction: %w: %q", ErrUnknownState, current.State)
	}
	return state.instruction, nil
}
