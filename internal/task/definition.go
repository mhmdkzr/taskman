package task

type (
	reducer   func(*Task, Event) error
	condition func(Task, Event) bool
)

type route struct {
	when condition
	to   State
}

type transition struct {
	to     State
	reduce reducer
	guard  condition
	routes []route
}

type stateDefinition struct {
	instruction Instruction
	terminal    bool
	on          map[EventKind]transition
}

type definition struct {
	initial State
	states  map[State]stateDefinition
	global  map[EventKind]transition
}
