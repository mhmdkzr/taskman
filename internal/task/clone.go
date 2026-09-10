package task

import "maps"

// Clone copies every mutable map, slice, and pointer owned by Task.
func (t Task) Clone() Task {
	out := t
	out.Labels = maps.Clone(t.Labels)
	out.References = append([]string(nil), t.References...)
	if t.Verifications != nil {
		out.Verifications = make([]Verification, len(t.Verifications))
		for i, verification := range t.Verifications {
			out.Verifications[i] = verification
			out.Verifications[i].Checks = maps.Clone(verification.Checks)
		}
	}
	if t.Reviews != nil {
		out.Reviews = make([]Review, len(t.Reviews))
		for i, review := range t.Reviews {
			out.Reviews[i] = review
			out.Reviews[i].Findings = append([]Finding(nil), review.Findings...)
		}
	}
	out.HumanReviews = append([]HumanReview(nil), t.HumanReviews...)
	out.SpecificationReviews = append([]SpecificationReview(nil), t.SpecificationReviews...)
	if t.Git.Commit != nil {
		out.Git.Commit = new(*t.Git.Commit)
	}
	if t.Blocked != nil {
		out.Blocked = new(*t.Blocked)
	}
	return out
}
