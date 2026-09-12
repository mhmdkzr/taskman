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
	Task Task `yaml:"task" json:"task"`
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
	ID             uuid.UUID       `yaml:"id"                        json:"id"`
	Definition     TaskDefinition  `yaml:"definition"                json:"definition"`
	StateHistory   []StateChange   `yaml:"state-history"             json:"state-history"`
	Specification  *Specification  `yaml:"specification,omitempty"   json:"specification,omitempty"`
	Implementation *Implementation `yaml:"implementation,omitempty"  json:"implementation,omitempty"`
	Blocked        *Blockage       `yaml:"blocked,omitempty"         json:"blocked,omitempty"`
	Abandoned      *Abandonment    `yaml:"abandoned,omitempty"       json:"abandoned,omitempty"`
}

// StateChange is one entry in a task's append-only state history.
type StateChange struct {
	State TaskState `yaml:"state" json:"state"`
	At    time.Time `yaml:"at"    json:"at"`
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

type TaskDefinition struct {
	Title       string            `yaml:"title"          json:"title"`
	Description string            `yaml:"description"    json:"description"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

type Specification struct {
	Plan   string              `yaml:"plan"   json:"plan"`
	Review ReviewConfiguration `yaml:"review" json:"review"`
}

type Implementation struct {
	Git          Git                 `yaml:"git"          json:"git"`
	Verification Verification        `yaml:"verification" json:"verification"`
	Review       ReviewConfiguration `yaml:"review"        json:"review"`
}

type Git struct {
	Worktree string      `yaml:"worktree"        json:"worktree"`
	Branch   string      `yaml:"branch"          json:"branch"`
	Commits  []GitCommit `yaml:"commits,omitempty" json:"commits,omitempty"`
	Merge    *GitMerge   `yaml:"merge,omitempty"   json:"merge,omitempty"`
}

type GitCommit struct {
	Hash    string    `yaml:"hash"    json:"hash"`
	Message string    `yaml:"message" json:"message"`
	At      time.Time `yaml:"at"      json:"at"`
}

// GitMerge refers to facts already owned by Git. The source is the last item
// in Git.Commits; Commit is the resulting commit on Target.
type GitMerge struct {
	Target string    `yaml:"target" json:"target"`
	Commit string    `yaml:"commit" json:"commit"`
	At     time.Time `yaml:"at"     json:"at"`
}

// Verification describes which build checks this task's workflow runs. A
// check that is false in Tests/Linters never gets a corresponding entry in
// Attempts.Checks; the bools are the sole source of truth for inclusion.
type Verification struct {
	Tests    TestConfiguration    `yaml:"tests"              json:"tests"`
	Linters  bool                 `yaml:"linters,omitempty"  json:"linters,omitempty"`
	AutoFix  AutoFix              `yaml:"auto-fix"           json:"auto-fix"`
	Attempts []VerificationResult `yaml:"attempts,omitempty" json:"attempts,omitempty"`
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
	Passed bool      `yaml:"passed"           json:"passed"`
	Checks Checks    `yaml:"checks"           json:"checks"`
	Output string    `yaml:"output,omitempty" json:"output,omitempty"`
	At     time.Time `yaml:"at"               json:"at"`
}

// Checks is one attempt's per-check outcomes. Its shape mirrors
// TestConfiguration plus Linters exactly - the same fixed set of checks a
// Verification can require - so an attempt can only ever report a check
// that's a real field here, and every required check must be reported: a
// mismatch either way (reported but not required, or required but missing)
// is rejected by validate.
type Checks struct {
	Unit        CheckResult `yaml:"unit,omitempty"        json:"unit,omitempty"`
	Integration CheckResult `yaml:"integration,omitempty" json:"integration,omitempty"`
	EndToEnd    CheckResult `yaml:"end-to-end,omitempty"  json:"end-to-end,omitempty"`
	Linters     CheckResult `yaml:"linters,omitempty"     json:"linters,omitempty"`
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
	Unit        bool `yaml:"unit,omitempty"        json:"unit,omitempty"`
	Integration bool `yaml:"integration,omitempty" json:"integration,omitempty"`
	EndToEnd    bool `yaml:"end-to-end,omitempty"  json:"end-to-end,omitempty"`
}

// ReviewConfiguration is always present on Specification and Implementation;
// each of Agent and Human decides its own inclusion via Required.
type ReviewConfiguration struct {
	Agent AgentReviewConfiguration `yaml:"agent-review" json:"agent-review"`
	Human HumanReviewConfiguration `yaml:"human-review" json:"human-review"`
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
	Required    bool                `yaml:"required"           json:"required"`
	UseSubagent bool                `yaml:"use-subagent,omitempty" json:"use-subagent,omitempty"`
	AutoFix     AutoFix             `yaml:"auto-fix"           json:"auto-fix"`
	Results     []AgentReviewResult `yaml:"results,omitempty"  json:"results,omitempty"`
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
	Required bool                `yaml:"required"          json:"required"`
	AutoFix  AutoFix             `yaml:"auto-fix"          json:"auto-fix"`
	Results  []HumanReviewResult `yaml:"results,omitempty" json:"results,omitempty"`
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
	Approved bool      `yaml:"approved"           json:"approved"`
	Comment  string    `yaml:"comment,omitempty"  json:"comment,omitempty"`
	Findings []Finding `yaml:"findings,omitempty" json:"findings,omitempty"`
	At       time.Time `yaml:"at"                 json:"at"`
}

type HumanReviewResult struct {
	Approved bool      `yaml:"approved"          json:"approved"`
	Comment  string    `yaml:"comment,omitempty" json:"comment,omitempty"`
	At       time.Time `yaml:"at"                json:"at"`
}

type Finding struct {
	Location string `yaml:"location" json:"location"`
	Detail   string `yaml:"detail"   json:"detail"`
}

// Blockage describes a paused, non-terminal task: the workflow is stuck at
// Stage for Reason pending outside intervention.
type Blockage struct {
	Stage  string `yaml:"stage"  json:"stage"`
	Reason string `yaml:"reason" json:"reason"`
}

// Abandonment is the task's one non-success terminal outcome. Reason is
// free text - the failure modes that lead here (infeasible, no longer
// needed, superseded, ...) are not branches the workflow treats
// differently, so there is no separate kind/taxonomy to maintain.
type Abandonment struct {
	Reason string    `yaml:"reason" json:"reason"`
	At     time.Time `yaml:"at"     json:"at"`
}

type AutoFix struct {
	Enabled     bool `yaml:"enabled"                json:"enabled"`
	MaxRounds   int  `yaml:"max-rounds,omitempty"   json:"max-rounds,omitempty"`
	UseSubagent bool `yaml:"use-subagent,omitempty" json:"use-subagent,omitempty"`
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

	switch state {
	case StateSpecificationReview, StateImplement, StateVerify,
		StateFixVerificationFailure, StateFixAutomatedReviewFindings,
		StateAutomatedReview, StateCommit, StateHumanReview,
		StateFixHumanReviewFindings, StateMerge, StateCompleted:
		if t.Specification == nil {
			return fmt.Errorf("state %s requires a specification", state)
		}
	}
	switch state {
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
