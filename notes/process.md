# The Process

## Discovery
Finding what needs to be done, could be based on code, system behavior, or an interview with user to capture their intent. Could also be a bug which needs to get fixed.

## Definition - The What and Why
The outcome of the discovery process. Value, budget, outcome risk, and deadline would be specified here, at goal level.

## Prioritization - The When
Weighing the importance and urgency of the discovered work.

## Specification - The How
Finding out the implementation design and details. The unit of work would be derived from here. Complexity, effort and execution risk must be specified here, at task level.

## Assignment - Agent Autonomy and Human Involvement
Deciding on level of human involvement, level of agent autonomy and agent model required for the work. Derived from the complexity and risk.

## Implementation
Making the code changes based on the specification. Costs would be tracked. If the specification itself turns out to be wrong or unworkable — not just the code — this exits directly back to Specification rather than consuming Verification's retry cap.

## Verification
Automated verification, including lints, static code analysis, tests, and automated reviews. Results in a feedback loop to the implementation step with a retry cap. Costs would be tracked.

## Review
PR creation and human review. Can result in a feedback loop to previous steps. Ends in a commit.

## Merge
Merge the PR into the codebase.

## Deployment
Deploy the code to the staging environment.

## Release
Promote a staged deployment to production. Kept distinct from Deployment since production promotion may need its own approval, canary period, or additional sign-off — particularly for anything carrying elevated risk from Definition or Specification.

## Monitoring
Monitoring the behavior of the deployed system. Can result in new discoveries. Costs would be tracked.

## Feedback and Calibration
Learning from results, and finding out how good the estimations were, so we can improve future estimates where possible.
