package loop

type State string

const (
	// Finding what needs to be done, could be based on code, system behavior,
	// or an interview with user to capture their intent. Could also be a bug
	// which needs to get fixed.
	Discovery State = "discovery"

	// The What and Why - The outcome of the discovery process. Value, budget and
	// outcome risk would be specified here, at goal level.
	Definition State = "definition"

	// The When - Weighting the importance and urgency of the discovered work.
	Prioritization State = "prioritization"

	// The How - Finding out the implementation design and details. The unit of
	// work would be derived from here. Complexity, effort and execution risk
	// must be specified here, at task level.
	Specification State = "specification"

	// Agent Autonomy and Human Involvement - Deciding on level of human involvement,
	// level of agent autonomy and agent model required for the work. Derived from the
	// complexity and risk.
	Assignment State = "assignment"

	// Making the code changes based on the specification. Costs would be tracked.
	Implementation State = "implementation"

	// Automated verification, including lints, static code analysis, tests, and
	// automated reviews. Results in a feedback loop to the implementation step
	// with a retry cap. Costs would be tracked.
	Verification State = "verification"

	// PR creation and human review. Can result in a feedback loop to previous steps.
	// Ends in a commit.
	Review State = "review"

	// Merge the PR into the codebase.
	Merge State = "merge"

	// Deploy the code to the staging environment.
	Deployment State = "deployment"

	// Monitoring the behavior of the deployed system. Can result in new discoveries.
	// Costs would be tracked.
	Monitoring State = "monitoring"

	// The feedback - Learning from results, and finding out how good the estimations
	// were, so we can improve future estimates where possible.
	Calibration State = "calibration"
)
