package task

import "maps"

// Clone deep-copies every mutable slice, map, and pointer owned by Task, so
// mutating the clone never affects the original. Apply relies on this to
// guarantee current is never observed as partially mutated.
func (t Task) Clone() Task {
	out := t
	out.StateHistory = append([]StateChange(nil), t.StateHistory...)
	out.Definition.Labels = maps.Clone(t.Definition.Labels)
	if t.Specification != nil {
		spec := *t.Specification
		spec.Review = t.Specification.Review.clone()
		out.Specification = &spec
	}
	if t.Implementation != nil {
		impl := *t.Implementation
		impl.Git.Commits = append([]GitCommit(nil), t.Implementation.Git.Commits...)
		if t.Implementation.Git.Merge != nil {
			merge := *t.Implementation.Git.Merge
			impl.Git.Merge = &merge
		}
		// VerificationResult.Checks is a plain struct (no maps/pointers), so
		// copying the slice's elements is already a full copy.
		impl.Verification.Attempts = append([]VerificationResult(nil), t.Implementation.Verification.Attempts...)
		impl.Review = t.Implementation.Review.clone()
		out.Implementation = &impl
	}
	if t.Blocked != nil {
		blocked := *t.Blocked
		out.Blocked = &blocked
	}
	if t.Abandoned != nil {
		abandoned := *t.Abandoned
		out.Abandoned = &abandoned
	}
	return out
}

func (r ReviewConfiguration) clone() ReviewConfiguration {
	out := r
	if r.Agent.Results != nil {
		out.Agent.Results = make([]AgentReviewResult, len(r.Agent.Results))
		for i, result := range r.Agent.Results {
			out.Agent.Results[i] = result
			out.Agent.Results[i].Findings = append([]Finding(nil), result.Findings...)
		}
	}
	out.Human.Results = append([]HumanReviewResult(nil), r.Human.Results...)
	return out
}
