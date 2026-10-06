package task

import "fmt"

type (
	reducer   func(*Task, TaskEvent) error
	condition func(Task, TaskEvent) bool
	// resolver computes a transition's destination from the pre- and
	// post-reduce tasks, for the rare transition whose destination isn't one
	// of a small fixed set of states knowable at compile time (see
	// resumeToPriorState and resolveAmendedSpecification). It receives
	// current (before the event) as well as next (after reduce) because some
	// destinations depend on what the event changed, not just the result.
	resolver func(current, next Task, event TaskEvent) (TaskState, error)
)

// route is one candidate destination of a transition with no fixed `to`;
// routes are tried in order and the first with a nil or true when wins.
type route struct {
	when condition
	to   TaskState
}

// transition is one event's effect from a given state (or globally). guard
// is checked against the pre-reduce task; reduce then runs on a clone;
// resolve (which receives both the pre- and post-reduce tasks), or else
// routes (or the fixed to), is evaluated against the post-reduce clone.
type transition struct {
	to      TaskState
	reduce  reducer
	guard   condition
	routes  []route
	resolve resolver
}

func (tr transition) destination(current, next Task, event TaskEvent) (TaskState, error) {
	if tr.to != "" {
		return tr.to, nil
	}
	if tr.resolve != nil {
		return tr.resolve(current, next, event)
	}
	for _, candidate := range tr.routes {
		if candidate.when == nil || candidate.when(next, event) {
			return candidate.to, nil
		}
	}
	return "", fmt.Errorf("%w: state=%s event=%s", errNoRoute, next.State(), event.Kind())
}

// stateDefinition is one state's workflow behavior: what an agent should do
// while the task sits there, whether the workflow ends there, and which
// events it accepts.
type stateDefinition struct {
	instruction Instruction
	terminal    bool
	on          map[EventKind]transition
}

// definition is the compiled workflow: every state's behavior, plus
// transitions (global) that apply regardless of the current state.
type definition struct {
	initial TaskState
	states  map[TaskState]stateDefinition
	global  map[EventKind]transition
}
