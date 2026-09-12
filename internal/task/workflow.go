package task

// workflow is the single compiled source of truth for both Apply's
// transitions and Instruction's per-state projection.
var workflow = definition{
	initial: StateSpecify,
	global: map[EventKind]transition{
		EventEscalated: {
			to:     StateBlocked,
			guard:  taskIsNotBlocked,
			reduce: recordEscalation,
		},
		EventAbandoned: {
			to:     StateAbandoned,
			reduce: recordAbandonment,
		},
	},
	states: map[TaskState]stateDefinition{
		StateSpecify: {
			instruction: Instruction{
				State:  StateSpecify,
				Action: InstructionDispatch,
			},
			on: map[EventKind]transition{
				EventSpecificationSubmitted: {
					reduce: recordSpecification,
					routes: []route{
						{when: specReviewRequired, to: StateSpecificationReview},
						{to: StateImplement},
					},
				},
			},
		},
		StateSpecificationReview: {
			instruction: Instruction{
				State:  StateSpecificationReview,
				Action: InstructionWait,
			},
			on: map[EventKind]transition{
				EventSpecificationReviewHumanApproved: {
					guard:  specHumanReviewRequired,
					reduce: recordSpecificationReviewHumanApproved,
					to:     StateImplement,
				},
				EventSpecificationReviewHumanRejected: {
					guard:  specHumanReviewRequired,
					reduce: recordSpecificationReviewHumanRejected,
					to:     StateSpecify,
				},
				EventSpecificationReviewAgentApproved: {
					guard:  specAgentReviewRequired,
					reduce: recordSpecificationReviewAgentApproved,
					routes: []route{
						{when: specHumanReviewRequired, to: StateSpecificationReview},
						{to: StateImplement},
					},
				},
				EventSpecificationReviewAgentRejected: {
					guard:  specAgentReviewRequired,
					reduce: recordSpecificationReviewAgentRejected,
					to:     StateSpecify,
				},
			},
		},
		StateImplement: {
			instruction: Instruction{
				State:  StateImplement,
				Action: InstructionDispatch,
			},
			on: map[EventKind]transition{
				EventImplementationCompleted: {
					reduce: recordImplementation,
					routes: []route{
						{when: verificationRequired, to: StateVerify},
						{when: implAgentReviewRequired, to: StateAutomatedReview},
						{to: StateCommit},
					},
				},
			},
		},
		StateVerify: {
			instruction: Instruction{
				State:  StateVerify,
				Action: InstructionRun,
			},
			on: verificationTransitions(StateFixVerificationFailure),
		},
		StateFixVerificationFailure: {
			instruction: Instruction{
				State:  StateFixVerificationFailure,
				Action: InstructionDispatch,
			},
			on: verificationTransitions(StateFixVerificationFailure),
		},
		StateFixAutomatedReviewFindings: {
			instruction: Instruction{
				State:  StateFixAutomatedReviewFindings,
				Action: InstructionDispatch,
			},
			on: verificationTransitions(StateFixVerificationFailure),
		},
		StateAutomatedReview: {
			instruction: Instruction{
				State:  StateAutomatedReview,
				Action: InstructionDispatch,
			},
			on: map[EventKind]transition{
				EventImplementationReviewAgentApproved: {
					guard:  implAgentReviewRequired,
					reduce: recordImplementationReviewAgentApproved,
					to:     StateCommit,
				},
				EventImplementationReviewAgentRejected: {
					guard:  implAgentReviewRequired,
					reduce: recordImplementationReviewAgentRejected,
					to:     StateFixAutomatedReviewFindings,
				},
			},
		},
		StateCommit: {
			instruction: Instruction{
				State:  StateCommit,
				Action: InstructionDispatch,
			},
			on: map[EventKind]transition{
				EventCommitRecorded: {
					reduce: recordCommit,
					routes: []route{
						{when: implHumanReviewRequired, to: StateHumanReview},
						{to: StateMerge},
					},
				},
			},
		},
		StateHumanReview: {
			instruction: Instruction{
				State:  StateHumanReview,
				Action: InstructionWait,
			},
			on: map[EventKind]transition{
				EventImplementationReviewHumanApproved: {
					guard:  implHumanReviewRequired,
					reduce: recordImplementationReviewHumanApproved,
					to:     StateMerge,
				},
				EventImplementationReviewHumanRejected: {
					guard:  implHumanReviewRequired,
					reduce: recordImplementationReviewHumanRejected,
					to:     StateFixHumanReviewFindings,
				},
			},
		},
		StateFixHumanReviewFindings: {
			instruction: Instruction{
				State:  StateFixHumanReviewFindings,
				Action: InstructionDispatch,
			},
			on: map[EventKind]transition{
				EventVerificationPassed: {
					reduce: recordVerificationPassed,
					to:     StateCommit,
				},
				EventVerificationFailed: {
					reduce: recordVerificationFailed,
					to:     StateFixHumanReviewFindings,
				},
			},
		},
		StateMerge: {
			instruction: Instruction{
				State:  StateMerge,
				Action: InstructionRun,
			},
			on: map[EventKind]transition{
				EventMergeCompleted: {
					reduce: recordMerge,
					to:     StateCompleted,
				},
			},
		},
		StateBlocked: {
			instruction: Instruction{
				State:  StateBlocked,
				Action: InstructionWait,
			},
		},
		StateCompleted: {
			instruction: Instruction{
				State:  StateCompleted,
				Action: InstructionDone,
			},
			terminal: true,
		},
		StateAbandoned: {
			instruction: Instruction{
				State:  StateAbandoned,
				Action: InstructionDone,
			},
			terminal: true,
		},
	},
}

// verificationTransitions is shared by every state that dispatches
// verification and retries on failure: StateVerify itself and the two
// "fix and reverify" states that loop back through the same checks.
func verificationTransitions(onFailure TaskState) map[EventKind]transition {
	return map[EventKind]transition{
		EventVerificationPassed: {
			reduce: recordVerificationPassed,
			routes: []route{
				{when: implAgentReviewRequired, to: StateAutomatedReview},
				{to: StateCommit},
			},
		},
		EventVerificationFailed: {
			reduce: recordVerificationFailed,
			to:     onFailure,
		},
	}
}

func specAgentReviewRequired(t Task, _ TaskEvent) bool {
	return t.Specification != nil && agentReviewRequired(t.Specification.Review)
}

func specHumanReviewRequired(t Task, _ TaskEvent) bool {
	return t.Specification != nil && humanReviewRequired(t.Specification.Review)
}

func specReviewRequired(t Task, _ TaskEvent) bool {
	return t.Specification != nil && reviewRequired(t.Specification.Review)
}

func implAgentReviewRequired(t Task, _ TaskEvent) bool {
	return t.Implementation != nil && agentReviewRequired(t.Implementation.Review)
}

func implHumanReviewRequired(t Task, _ TaskEvent) bool {
	return t.Implementation != nil && humanReviewRequired(t.Implementation.Review)
}

func verificationRequired(t Task, _ TaskEvent) bool {
	return t.Implementation != nil && t.Implementation.Verification.required()
}

func taskIsNotBlocked(t Task, _ TaskEvent) bool {
	return t.State() != StateBlocked
}
