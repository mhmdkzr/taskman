package task

var workflow = mustBuild(definition{
	initial: StateSpecify,
	global: map[EventKind]transition{
		EventEscalated: {
			to:     StateBlocked,
			reduce: recordEscalation,
			guard:  taskIsNotBlocked,
		},
		EventAbandoned: {
			to:     StateAbandoned,
			reduce: recordAbandonment,
		},
	},
	states: map[State]stateDefinition{
		StateSpecify:                    specifyStateDefinition,
		StateSpecificationReview:        specificationReviewStateDefinition,
		StateImplement:                  implementStateDefinition,
		StateVerify:                     verifyStateDefinition,
		StateFixVerificationFailure:     fixVerificationFailureStateDefinition,
		StateFixAutomatedReviewFindings: fixAutomatedReviewFindingsStateDefinition,
		StateAutomatedReview:            automatedReviewStateDefinition,
		StateCommit:                     commitStateDefinition,
		StateHumanReview:                humanReviewStateDefinition,
		StateFixHumanReviewFindings:     fixHumanReviewFindingsStateDefinition,
		StateMerge:                      mergeStateDefinition,
		StateBlocked:                    blockedStateDefinition,
		StateCompleted:                  completedStateDefinition,
		StateAbandoned:                  abandonedStateDefinition,
	},
})

var specifyStateDefinition = stateDefinition{
	instruction: Instruction{
		Kind:   InstructionSpecify,
		Action: InstructionDispatch,
	},
	on: map[EventKind]transition{
		EventSpecificationSubmitted: {
			to:     StateSpecificationReview,
			reduce: recordSpecification,
		},
	},
}

var specificationReviewStateDefinition = stateDefinition{
	instruction: Instruction{
		Kind:   InstructionSpecificationReview,
		Action: InstructionWait,
	},
	on: map[EventKind]transition{
		EventSpecificationApproved: {
			to:     StateImplement,
			reduce: recordSpecificationApproval,
		},
		EventSpecificationRejected: {
			to:     StateSpecify,
			reduce: recordSpecificationRejection,
		},
	},
}

var implementStateDefinition = stateDefinition{
	instruction: Instruction{
		Kind:   InstructionImplement,
		Action: InstructionDispatch,
	},
	on: map[EventKind]transition{
		EventImplementationCompleted: {to: StateVerify},
	},
}

var verifyStateDefinition = stateDefinition{
	instruction: Instruction{
		Kind:   InstructionVerify,
		Action: InstructionRun,
	},
	on: map[EventKind]transition{
		EventVerificationReported: {
			reduce: recordVerification,
			routes: []route{
				{when: verificationPassed, to: StateAutomatedReview},
				{to: StateFixVerificationFailure},
			},
		},
	},
}

var fixVerificationFailureStateDefinition = stateDefinition{
	instruction: Instruction{
		Kind:   InstructionFixVerificationFailure,
		Action: InstructionDispatch,
	},
	on: map[EventKind]transition{
		EventVerificationReported: {
			reduce: recordVerification,
			routes: []route{
				{when: verificationPassed, to: StateAutomatedReview},
				{to: StateFixVerificationFailure},
			},
		},
	},
}

var fixAutomatedReviewFindingsStateDefinition = stateDefinition{
	instruction: Instruction{
		Kind:   InstructionFixAutomatedReviewFindings,
		Action: InstructionDispatch,
	},
	on: map[EventKind]transition{
		EventVerificationReported: {
			reduce: recordVerification,
			routes: []route{
				{when: verificationPassed, to: StateAutomatedReview},
				{to: StateFixVerificationFailure},
			},
		},
	},
}

var automatedReviewStateDefinition = stateDefinition{
	instruction: Instruction{
		Kind:   InstructionAutomatedReview,
		Action: InstructionDispatch,
	},
	on: map[EventKind]transition{
		EventAutomatedReviewRecorded: {
			reduce: recordAutomatedReview,
			routes: []route{
				{when: automatedReviewApproved, to: StateCommit},
				{when: automatedReviewLimitReached, to: StateBlocked},
				{to: StateFixAutomatedReviewFindings},
			},
		},
	},
}

var commitStateDefinition = stateDefinition{
	instruction: Instruction{
		Kind:   InstructionCommit,
		Action: InstructionDispatch,
	},
	on: map[EventKind]transition{
		EventCommitRecorded: {
			reduce: recordCommit,
			routes: []route{
				{when: autoApprovedOnTrunk, to: StateCompleted},
				{when: autoApproved, to: StateMerge},
				{to: StateHumanReview},
			},
		},
	},
}

var humanReviewStateDefinition = stateDefinition{
	instruction: Instruction{
		Kind:   InstructionHumanReview,
		Action: InstructionWait,
	},
	on: map[EventKind]transition{
		EventHumanReviewApproved: {
			reduce: recordHumanApproval,
			routes: []route{
				{when: worksOnTrunk, to: StateCompleted},
				{to: StateMerge},
			},
		},
		EventHumanReviewRejected: {
			to:     StateFixHumanReviewFindings,
			reduce: recordHumanRejection,
		},
	},
}

var fixHumanReviewFindingsStateDefinition = stateDefinition{
	instruction: Instruction{
		Kind:   InstructionFixHumanReviewFindings,
		Action: InstructionDispatch,
	},
	on: map[EventKind]transition{
		EventVerificationReported: {
			reduce: recordVerification,
			routes: []route{
				{when: verificationPassed, to: StateCommit},
				{to: StateFixHumanReviewFindings},
			},
		},
	},
}

var mergeStateDefinition = stateDefinition{
	instruction: Instruction{
		Kind:   InstructionMerge,
		Action: InstructionRun,
	},
	on: map[EventKind]transition{
		EventMergeCompleted: {
			to:     StateCompleted,
			reduce: recordMerge,
		},
	},
}

var blockedStateDefinition = stateDefinition{
	instruction: Instruction{
		Kind:   InstructionBlocked,
		Action: InstructionWait,
	},
}

var completedStateDefinition = stateDefinition{
	instruction: Instruction{
		Kind:   InstructionCompleted,
		Action: InstructionDone,
	},
	terminal: true,
}

var abandonedStateDefinition = stateDefinition{
	instruction: Instruction{
		Kind:   InstructionAbandoned,
		Action: InstructionDone,
	},
	terminal: true,
}

// InitialState is the starting point for every newly created task.
func InitialState() State {
	return workflow.initial
}
