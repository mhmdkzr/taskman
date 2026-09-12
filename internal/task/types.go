// Package task is taskman's pure functional core: the Task aggregate, the
// closed set of TaskEvent types, the declarative workflow, Apply, and the
// per-state Instruction projection. It performs no filesystem, Git, clock,
// logging, CLI, or MCP work - every event carries its own At, supplied by
// the caller.
package task

import (
	"fmt"
	"time"
	"uuid"
)

// TaskDocument is the top-level shape of a rendered task (e.g. for a human-
// facing view) - not the on-disk format, which is an event log; see
// task/store.
type TaskDocument struct {
	Task Task `json:"task" yaml:"task"`
}

func (s TaskState) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *TaskState) UnmarshalText(text []byte) error {
	parsed, err := ParseTaskState(string(text))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// Task contains the task's identity and the portions of the workflow that
// have been configured or recorded. Specification and Implementation are the
// only optional (pointer) parts left: their nil-ness is purely temporal ("not
// reached yet"), since once set they are never cleared. Every other part of
// the workflow - which reviews run, which verification checks run, whether
// auto-fix runs - is represented by a leaf bool that is always present, so
// "excluded" and "not yet decided" are never the same zero value.
type Task struct {
	ID             uuid.UUID       `json:"id"                       yaml:"id"`
	Definition     TaskDefinition  `json:"definition"               yaml:"definition"`
	StateHistory   []StateChange   `json:"state-history"            yaml:"state-history"`
	Specification  *Specification  `json:"specification,omitempty"  yaml:"specification,omitempty"`
	Implementation *Implementation `json:"implementation,omitempty" yaml:"implementation,omitempty"`
	Blocked        *Blockage       `json:"blocked,omitempty"        yaml:"blocked,omitempty"`
	Abandoned      *Abandonment    `json:"abandoned,omitempty"      yaml:"abandoned,omitempty"`
}

// StateChange is one entry in a task's append-only state history.
type StateChange struct {
	State TaskState `json:"state" yaml:"state"`
	At    time.Time `json:"at"    yaml:"at"`
}

type TaskDefinition struct {
	Title       string            `json:"title"            yaml:"title"`
	Description string            `json:"description"      yaml:"description"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type Specification struct {
	Plan   string              `json:"plan"   yaml:"plan"`
	Review ReviewConfiguration `json:"review" yaml:"review"`
}

type Implementation struct {
	Git          Git                 `json:"git"          yaml:"git"`
	Verification Verification        `json:"verification" yaml:"verification"`
	Review       ReviewConfiguration `json:"review"       yaml:"review"`
}

type Git struct {
	Worktree string      `json:"worktree"          yaml:"worktree"`
	Branch   string      `json:"branch"            yaml:"branch"`
	Commits  []GitCommit `json:"commits,omitempty" yaml:"commits,omitempty"`
	Merge    *GitMerge   `json:"merge,omitempty"   yaml:"merge,omitempty"`
}

type GitCommit struct {
	Hash    string    `json:"hash"    yaml:"hash"`
	Message string    `json:"message" yaml:"message"`
	At      time.Time `json:"at"      yaml:"at"`
}

// GitMerge refers to facts already owned by Git. The source is the last item
// in Git.Commits; Commit is the resulting commit on Target.
type GitMerge struct {
	Target string    `json:"target" yaml:"target"`
	Commit string    `json:"commit" yaml:"commit"`
	At     time.Time `json:"at"     yaml:"at"`
}

// Verification describes which build checks this task's workflow runs. A
// check that is false in Tests/Linters never gets a corresponding entry in
// Attempts.Checks; the bools are the sole source of truth for inclusion.
type Verification struct {
	Tests    TestConfiguration    `json:"tests"              yaml:"tests"`
	Linters  bool                 `json:"linters,omitempty"  yaml:"linters,omitempty"`
	AutoFix  AutoFix              `json:"auto-fix"           yaml:"auto-fix"`
	Attempts []VerificationResult `json:"attempts,omitempty" yaml:"attempts,omitempty"`
}

// validate enforces the implementation's configuration invariants. The key
// one: every review gate requires verification. A rejected review always
// loops back through a verification attempt (see StateFixAutomatedReviewFindings
// and StateFixHumanReviewFindings), so a gate without any configured check
// would strand the task in a fix state with no valid way out.
func (i Implementation) validate() error {
	if err := i.Verification.validate(); err != nil {
		return fmt.Errorf("verification: %w", err)
	}
	if reviewRequired(i.Review) && !i.Verification.required() {
		return fmt.Errorf("review gates require at least one verification check")
	}
	if err := i.Review.validate(); err != nil {
		return fmt.Errorf("review: %w", err)
	}
	return nil
}

// required reports whether any verification check is configured to run.
func (v Verification) required() bool {
	return v.Tests.Unit || v.Tests.Integration || v.Tests.EndToEnd || v.Linters
}

func (v Verification) validate() error {
	if !v.required() {
		if v.AutoFix != (AutoFix{}) || len(v.Attempts) > 0 {
			return fmt.Errorf("verification is not required but has configuration or attempts")
		}
		return nil
	}
	for _, attempt := range v.Attempts {
		if err := attempt.Checks.validate(v.Tests, v.Linters); err != nil {
			return err
		}
	}
	return nil
}

type VerificationResult struct {
	Passed bool      `json:"passed"           yaml:"passed"`
	Checks Checks    `json:"checks"           yaml:"checks"`
	Output string    `json:"output,omitempty" yaml:"output,omitempty"`
	At     time.Time `json:"at"               yaml:"at"`
}

// Checks is one attempt's per-check outcomes. Its shape mirrors
// TestConfiguration plus Linters exactly - the same fixed set of checks a
// Verification can require - so an attempt can only ever report a check
// that's a real field here, and every required check must be reported: a
// mismatch either way (reported but not required, or required but missing)
// is rejected by validate.
type Checks struct {
	Unit        CheckResult `json:"unit,omitempty"        yaml:"unit,omitempty"`
	Integration CheckResult `json:"integration,omitempty" yaml:"integration,omitempty"`
	EndToEnd    CheckResult `json:"end-to-end,omitempty"  yaml:"end-to-end,omitempty"`
	Linters     CheckResult `json:"linters,omitempty"     yaml:"linters,omitempty"`
}

func (c Checks) validate(tests TestConfiguration, linters bool) error {
	if (c.Unit != "") != tests.Unit {
		return fmt.Errorf("unit check reported=%v, required=%v", c.Unit != "", tests.Unit)
	}
	if (c.Integration != "") != tests.Integration {
		return fmt.Errorf("integration check reported=%v, required=%v", c.Integration != "", tests.Integration)
	}
	if (c.EndToEnd != "") != tests.EndToEnd {
		return fmt.Errorf("end-to-end check reported=%v, required=%v", c.EndToEnd != "", tests.EndToEnd)
	}
	if (c.Linters != "") != linters {
		return fmt.Errorf("linters check reported=%v, required=%v", c.Linters != "", linters)
	}
	return nil
}

// CheckResult is one check's outcome. It is a string, not a bool, because
// its zero value has to mean "not applicable to this task" distinctly from
// either real outcome - a zero-value bool would silently read as "failed"
// for a check that was never even required, which is exactly the kind of
// zero-value ambiguity this type exists to avoid. Contrast with
// Tests/Linters/Required/Approved elsewhere, which stay bool because their
// zero value (false) already is the correct, unambiguous default.
type CheckResult string

const (
	CheckOK    CheckResult = "ok"
	CheckError CheckResult = "error"
)

type TestConfiguration struct {
	Unit        bool `json:"unit,omitempty"        yaml:"unit,omitempty"`
	Integration bool `json:"integration,omitempty" yaml:"integration,omitempty"`
	EndToEnd    bool `json:"end-to-end,omitempty"  yaml:"end-to-end,omitempty"`
}

// ReviewConfiguration is always present on Specification and Implementation;
// each of Agent and Human decides its own inclusion via Required.
type ReviewConfiguration struct {
	Agent AgentReviewConfiguration `json:"agent-review" yaml:"agent-review"`
	Human HumanReviewConfiguration `json:"human-review" yaml:"human-review"`
}

func (r ReviewConfiguration) validate() error {
	if err := r.Agent.validate(); err != nil {
		return fmt.Errorf("agent-review: %w", err)
	}
	if err := r.Human.validate(); err != nil {
		return fmt.Errorf("human-review: %w", err)
	}
	return nil
}

// AgentReviewConfiguration is the gate for one automated-review stage.
// Required is the only field consulted to decide whether this stage runs;
// when it is false, UseSubagent, AutoFix, and Results must stay zero/empty -
// there is no separate way to say "excluded" versus "not yet configured".
type AgentReviewConfiguration struct {
	Required    bool                `json:"required"               yaml:"required"`
	UseSubagent bool                `json:"use-subagent,omitempty" yaml:"use-subagent,omitempty"`
	AutoFix     AutoFix             `json:"auto-fix"               yaml:"auto-fix"`
	Results     []AgentReviewResult `json:"results,omitempty"      yaml:"results,omitempty"`
}

func (a AgentReviewConfiguration) validate() error {
	if a.Required {
		return nil
	}
	if a.UseSubagent || a.AutoFix != (AutoFix{}) || len(a.Results) > 0 {
		return fmt.Errorf("not required but has configuration or results")
	}
	return nil
}

// HumanReviewConfiguration is the gate for one human-review stage. See
// AgentReviewConfiguration for the meaning of Required.
type HumanReviewConfiguration struct {
	Required bool                `json:"required"          yaml:"required"`
	AutoFix  AutoFix             `json:"auto-fix"          yaml:"auto-fix"`
	Results  []HumanReviewResult `json:"results,omitempty" yaml:"results,omitempty"`
}

func (h HumanReviewConfiguration) validate() error {
	if h.Required {
		return nil
	}
	if h.AutoFix != (AutoFix{}) || len(h.Results) > 0 {
		return fmt.Errorf("not required but has configuration or results")
	}
	return nil
}

type AgentReviewResult struct {
	Approved bool      `json:"approved"           yaml:"approved"`
	Comment  string    `json:"comment,omitempty"  yaml:"comment,omitempty"`
	Findings []Finding `json:"findings,omitempty" yaml:"findings,omitempty"`
	At       time.Time `json:"at"                 yaml:"at"`
}

type HumanReviewResult struct {
	Approved bool      `json:"approved"          yaml:"approved"`
	Comment  string    `json:"comment,omitempty" yaml:"comment,omitempty"`
	At       time.Time `json:"at"                yaml:"at"`
}

type Finding struct {
	Location string `json:"location" yaml:"location"`
	Detail   string `json:"detail"   yaml:"detail"`
}

// Blockage describes a paused, non-terminal task: the workflow is stuck at
// Stage for Reason pending outside intervention.
type Blockage struct {
	Stage  string `json:"stage"  yaml:"stage"`
	Reason string `json:"reason" yaml:"reason"`
}

// Abandonment is the task's one non-success terminal outcome. Reason is
// free text - the failure modes that lead here (infeasible, no longer
// needed, superseded, ...) are not branches the workflow treats
// differently, so there is no separate kind/taxonomy to maintain.
type Abandonment struct {
	Reason string    `json:"reason" yaml:"reason"`
	At     time.Time `json:"at"     yaml:"at"`
}

type AutoFix struct {
	Enabled     bool `json:"enabled"                yaml:"enabled"`
	MaxRounds   int  `json:"max-rounds,omitempty"   yaml:"max-rounds,omitempty"`
	UseSubagent bool `json:"use-subagent,omitempty" yaml:"use-subagent,omitempty"`
}

// NewTask constructs a task in the initial specify state.
func NewTask(id uuid.UUID, definition TaskDefinition, at time.Time) (Task, error) {
	if id == uuid.Nil() {
		return Task{}, fmt.Errorf("new task: id is required")
	}
	if definition.Description == "" {
		return Task{}, fmt.Errorf("new task: description is required")
	}
	t := Task{
		ID:           id,
		Definition:   definition,
		StateHistory: []StateChange{{State: StateSpecify, At: at}},
	}
	if err := t.Validate(); err != nil {
		return Task{}, fmt.Errorf("new task: %w", err)
	}
	return t, nil
}

// State returns the task's current position in the workflow: the last entry
// of StateHistory. It is derived, never stored independently - Apply appends
// to StateHistory rather than assigning a separate field.
func (t Task) State() TaskState {
	if len(t.StateHistory) == 0 {
		return ""
	}
	return t.StateHistory[len(t.StateHistory)-1].State
}

// Validate checks the task's workflow and data invariants.
func (t Task) Validate() error {
	if t.ID == uuid.Nil() || t.Definition.Description == "" {
		return fmt.Errorf("task identity and definition are required")
	}
	if len(t.StateHistory) == 0 {
		return fmt.Errorf("task has no recorded state")
	}
	state := t.State()
	if !state.valid() {
		return fmt.Errorf("unknown task state")
	}
	if state != StateBlocked && t.Blocked != nil {
		return fmt.Errorf("state %s cannot contain blocked data", state)
	}
	if state == StateBlocked && t.Blocked == nil {
		return fmt.Errorf("blocked state requires blocked data")
	}
	if state != StateAbandoned && t.Abandoned != nil {
		return fmt.Errorf("state %s cannot contain abandoned data", state)
	}
	if state == StateAbandoned && t.Abandoned == nil {
		return fmt.Errorf("abandoned state requires abandoned data")
	}

	switch state { //nolint:exhaustive // only these states require a specification; others are checked below.
	case StateSpecificationReview, StateImplement, StateVerify,
		StateFixVerificationFailure, StateFixAutomatedReviewFindings,
		StateAutomatedReview, StateCommit, StateHumanReview,
		StateFixHumanReviewFindings, StateMerge, StateCompleted:
		if t.Specification == nil {
			return fmt.Errorf("state %s requires a specification", state)
		}
	}
	switch state { //nolint:exhaustive // only these states require implementation; others are checked below.
	case StateVerify, StateFixVerificationFailure,
		StateFixAutomatedReviewFindings, StateAutomatedReview,
		StateCommit, StateHumanReview, StateFixHumanReviewFindings,
		StateMerge, StateCompleted:
		if t.Implementation == nil {
			return fmt.Errorf("state %s requires implementation data", state)
		}
	}
	if (state == StateHumanReview || state == StateMerge || state == StateCompleted) &&
		!t.hasCommit() {
		return fmt.Errorf("state %s requires a recorded commit", state)
	}
	if state == StateCompleted && !t.hasMerge() {
		return fmt.Errorf("completed state requires a recorded merge")
	}

	if t.Specification != nil {
		if err := t.Specification.Review.validate(); err != nil {
			return fmt.Errorf("specification review: %w", err)
		}
	}
	if t.Implementation != nil {
		if err := t.Implementation.validate(); err != nil {
			return fmt.Errorf("implementation: %w", err)
		}
	}
	return nil
}

func (t Task) hasCommit() bool {
	return t.Implementation != nil && len(t.Implementation.Git.Commits) > 0
}

func (t Task) hasMerge() bool {
	return t.Implementation != nil && t.Implementation.Git.Merge != nil
}
