// Package loop defines the lifecycle states used by the application.
package loop

type State string

const (
	// Discovery is finding what needs to be done, based on code, system behavior,
	// or an interview with user to capture their intent. Could also be a bug
	// which needs to get fixed.
	Discovery State = "discovery"

	// Definition is the What and Why: the outcome of the discovery process. Value, budget,
	// outcome risk, and deadline would be specified here, at goal level.
	Definition State = "definition"

	// Prioritization is the When: weighing the importance and urgency of the discovered work.
	Prioritization State = "prioritization"

	// Specification is the How: finding the implementation design and details. The unit of
	// work would be derived from here. Complexity, effort and execution risk
	// must be specified here, at task level.
	Specification State = "specification"

	// Assignment decides agent autonomy and human involvement, including the
	// level of agent autonomy and agent model required for the work. Derived from the
	// complexity and risk.
	Assignment State = "assignment"

	// Implementation makes the code changes based on the specification. Costs would be tracked.
	// If the specification itself turns out to be wrong or unworkable, this exits
	// directly back to Specification rather than consuming Verification's retry cap.
	Implementation State = "implementation"

	// Verification performs automated checks, including lints, static code analysis, tests, and
	// automated reviews. Results in a feedback loop to the implementation step
	// with a retry cap. Costs would be tracked.
	Verification State = "verification"

	// Review covers PR creation and human review. It can result in a feedback loop to previous steps.
	// Ends in a commit.
	Review State = "review"

	// Merge the PR into the codebase.
	Merge State = "merge"

	// Deployment deploys the code to the staging environment.
	Deployment State = "deployment"

	// Release promotes a staged deployment to production. It is distinct from Deployment
	// since production promotion may need its own approval, canary period, or
	// additional sign-off, particularly for anything carrying elevated risk from
	// Definition or Specification.
	Release State = "release"

	// Monitoring the behavior of the deployed system. Can result in new discoveries.
	// Costs would be tracked.
	Monitoring State = "monitoring"

	// Feedback learns from results and finds out how good the estimations were, so we
	// can improve future estimates where possible.
	Feedback State = "feedback"
)
