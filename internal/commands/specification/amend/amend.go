// Package amend owns the "specification amend" command: it revises a task's
// specification after it has been submitted but before its implementation is
// recorded. Every field is a patch - only the parts a caller explicitly
// supplies are applied over the task's current specification - so a targeted
// change (such as enabling auto-fix) never silently drops a gate.
//
// The patch types mirror task.Specification's own nested shape, and the CLI
// flags are specified's flags, so the MCP tool's input mirrors specified's and
// an agent can move between the two without translating field names.
package amend

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is amend's input. A nil field means "keep the current value"; a
// non-nil field overrides it. At least one field must be supplied.
type Request struct {
	ID   uuid.UUID `json:"id"             jsonschema:"the task whose specification is being amended"`
	Plan *string   `json:"plan,omitempty" jsonschema:"replacement plan; omit to keep the current plan"`

	Review               *ReviewPatch       `json:"review,omitempty"                jsonschema:"patch for the specification-review gates"`
	ImplementationReview *ReviewPatch       `json:"implementation-review,omitempty" jsonschema:"patch for the implementation-review gates"`
	Verification         *VerificationPatch `json:"verification,omitempty"          jsonschema:"patch for the verification checks"`
	Worktree             *WorktreePatch     `json:"worktree,omitempty"              jsonschema:"patch for the worktree policy"`
}

// ReviewPatch mirrors task.ReviewConfiguration with every leaf optional.
type ReviewPatch struct {
	Agent *AgentReviewPatch `json:"agent-review,omitempty" jsonschema:"the automated review gate"`
	Human *HumanReviewPatch `json:"human-review,omitempty" jsonschema:"the human review gate"`
}

// AgentReviewPatch mirrors task.AgentReviewConfiguration with every leaf optional.
type AgentReviewPatch struct {
	Required    *bool         `json:"required,omitempty"     jsonschema:"require an automated review"`
	UseSubagent *bool         `json:"use-subagent,omitempty" jsonschema:"run the automated review in a subagent"`
	AutoFix     *AutoFixPatch `json:"auto-fix,omitempty"     jsonschema:"the automated review's auto-fix policy"`
}

// HumanReviewPatch mirrors task.HumanReviewConfiguration with every leaf optional.
type HumanReviewPatch struct {
	Required *bool         `json:"required,omitempty" jsonschema:"require a human review"`
	AutoFix  *AutoFixPatch `json:"auto-fix,omitempty" jsonschema:"the human review's auto-fix policy"`
}

// AutoFixPatch mirrors task.AutoFix with every leaf optional.
type AutoFixPatch struct {
	Enabled     *bool `json:"enabled,omitempty"      jsonschema:"automatically fix findings"`
	MaxRounds   *int  `json:"max-rounds,omitempty"   jsonschema:"max auto-fix rounds"`
	UseSubagent *bool `json:"use-subagent,omitempty" jsonschema:"run auto-fix in a subagent"`
}

// VerificationPatch mirrors task.Verification with every leaf optional.
type VerificationPatch struct {
	Tests   *TestPatch    `json:"tests,omitempty"    jsonschema:"the test checks to require"`
	Linters *bool         `json:"linters,omitempty"  jsonschema:"require linters"`
	AutoFix *AutoFixPatch `json:"auto-fix,omitempty" jsonschema:"the verification auto-fix policy"`
}

// TestPatch mirrors task.TestConfiguration with every leaf optional.
type TestPatch struct {
	Unit        *bool `json:"unit,omitempty"        jsonschema:"require unit tests"`
	Integration *bool `json:"integration,omitempty" jsonschema:"require integration tests"`
	EndToEnd    *bool `json:"end-to-end,omitempty"  jsonschema:"require end-to-end tests"`
}

// WorktreePatch mirrors task.WorktreePolicy with every leaf optional.
type WorktreePatch struct {
	UseWorktree *bool   `json:"use-worktree,omitempty" jsonschema:"implement this task in a fresh worktree"`
	Worktree    *string `json:"worktree,omitempty"     jsonschema:"the worktree path to use, if use-worktree"`
	Branch      *string `json:"branch,omitempty"       jsonschema:"the branch name to use, if use-worktree"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	if r.Plan == nil && r.Review == nil && r.ImplementationReview == nil &&
		r.Verification == nil && r.Worktree == nil {
		return fmt.Errorf("at least one field to amend is required")
	}
	return nil
}

// Amend applies req's patch to the task's current specification and records
// the result as a SpecificationAmended event. The store's Apply re-derives the
// task's state from the change, so a spec whose review inputs changed runs
// its specification review again.
func Amend(ctx context.Context, st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("amend specification: %w", err)
	}
	current, err := st.Read(ctx, req.ID)
	if err != nil {
		return task.Task{}, fmt.Errorf("amend specification: %w", err)
	}
	if current.Specification == nil {
		return task.Task{}, fmt.Errorf("amend specification: task has no specification to amend")
	}

	spec := applyPatch(*current.Specification, req)
	// Clear review outcomes so the reducer is the sole authority on whether
	// they survive: it carries them forward only when the review inputs are
	// untouched.
	spec.Review.Agent.Results = nil
	spec.Review.Agent.Unblocks = nil
	spec.Review.Human.Results = nil
	spec.Review.Human.Unblocks = nil

	event := task.SpecificationAmended{Specification: spec, At: time.Now().UTC()}
	t, err := st.Append(ctx, req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("amend specification: %w", err)
	}
	return t, nil
}

func applyPatch(spec task.Specification, req Request) task.Specification {
	if req.Plan != nil {
		spec.Plan = *req.Plan
	}
	if req.Review != nil {
		spec.Review = req.Review.apply(spec.Review)
	}
	if req.ImplementationReview != nil {
		spec.ImplementationReview = req.ImplementationReview.apply(spec.ImplementationReview)
	}
	if req.Verification != nil {
		spec.Verification = req.Verification.apply(spec.Verification)
	}
	if req.Worktree != nil {
		spec.Worktree = req.Worktree.apply(spec.Worktree)
	}
	return spec
}

func (p ReviewPatch) apply(base task.ReviewConfiguration) task.ReviewConfiguration {
	if p.Agent != nil {
		setBool(&base.Agent.Required, p.Agent.Required)
		setBool(&base.Agent.UseSubagent, p.Agent.UseSubagent)
		if p.Agent.AutoFix != nil {
			base.Agent.AutoFix = p.Agent.AutoFix.apply(base.Agent.AutoFix)
		}
	}
	if p.Human != nil {
		setBool(&base.Human.Required, p.Human.Required)
		if p.Human.AutoFix != nil {
			base.Human.AutoFix = p.Human.AutoFix.apply(base.Human.AutoFix)
		}
	}
	return base
}

func (p AutoFixPatch) apply(base task.AutoFix) task.AutoFix {
	setBool(&base.Enabled, p.Enabled)
	setInt(&base.MaxRounds, p.MaxRounds)
	setBool(&base.UseSubagent, p.UseSubagent)
	return base
}

func (p VerificationPatch) apply(base task.Verification) task.Verification {
	if p.Tests != nil {
		setBool(&base.Tests.Unit, p.Tests.Unit)
		setBool(&base.Tests.Integration, p.Tests.Integration)
		setBool(&base.Tests.EndToEnd, p.Tests.EndToEnd)
	}
	setBool(&base.Linters, p.Linters)
	if p.AutoFix != nil {
		base.AutoFix = p.AutoFix.apply(base.AutoFix)
	}
	return base
}

func (p WorktreePatch) apply(base task.WorktreePolicy) task.WorktreePolicy {
	setBool(&base.UseWorktree, p.UseWorktree)
	setString(&base.Worktree, p.Worktree)
	setString(&base.Branch, p.Branch)
	return base
}

func (p ReviewPatch) empty() bool {
	return p.Agent == nil && p.Human == nil
}

func (p AgentReviewPatch) empty() bool {
	return p.Required == nil && p.UseSubagent == nil && p.AutoFix == nil
}

func (p HumanReviewPatch) empty() bool {
	return p.Required == nil && p.AutoFix == nil
}

func (p AutoFixPatch) empty() bool {
	return p.Enabled == nil && p.MaxRounds == nil && p.UseSubagent == nil
}

func (p VerificationPatch) empty() bool {
	return p.Tests == nil && p.Linters == nil && p.AutoFix == nil
}

func (p TestPatch) empty() bool {
	return p.Unit == nil && p.Integration == nil && p.EndToEnd == nil
}

func (p WorktreePatch) empty() bool {
	return p.UseWorktree == nil && p.Worktree == nil && p.Branch == nil
}

func setBool(dst *bool, src *bool) {
	if src != nil {
		*dst = *src
	}
}

func setInt(dst *int, src *int) {
	if src != nil {
		*dst = *src
	}
}

func setString(dst *string, src *string) {
	if src != nil {
		*dst = *src
	}
}
