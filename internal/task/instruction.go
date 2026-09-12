package task

import "slices"

// InstructionAction is what an agent should do while a task sits in a given
// state.
type InstructionAction string

const (
	// InstructionDispatch means an agent should act (implement, fix, review).
	InstructionDispatch InstructionAction = "dispatch"
	// InstructionRun means a deterministic check should be run (verify, merge).
	InstructionRun InstructionAction = "run"
	// InstructionWait means the task is paused for an event outside the
	// agent's control (a human's review, an escalation being resolved).
	InstructionWait InstructionAction = "wait"
	// InstructionDone means the workflow has ended; no further event applies.
	InstructionDone InstructionAction = "done"
)

// Instruction is the pure workflow projection of a task's current state:
// what should happen next, and the state it was derived from.
type Instruction struct {
	State  TaskState         `json:"state"`
	Action InstructionAction `json:"action"`
}

// Instruction returns what should happen next given t's current state. It is
// a pure function of State - the same state always yields the same
// instruction, regardless of how the task got there.
func (t Task) Instruction() Instruction {
	def, ok := workflow.states[t.State()]
	if !ok {
		return Instruction{State: t.State(), Action: InstructionDone}
	}
	return def.instruction
}

// ValidEvents returns the EventKinds t's current state accepts right now -
// the same guards Apply would enforce, evaluated without attempting a
// transition. It reads only per-state transitions, not the global
// escalate/abandon escape hatches available from every non-terminal state.
// The result is sorted for determinism.
func (t Task) ValidEvents() []EventKind {
	def, ok := workflow.states[t.State()]
	if !ok || def.terminal {
		return nil
	}
	kinds := make([]EventKind, 0, len(def.on))
	for kind, tr := range def.on {
		if tr.guard == nil || tr.guard(t, nil) {
			kinds = append(kinds, kind)
		}
	}
	slices.Sort(kinds)
	return kinds
}
