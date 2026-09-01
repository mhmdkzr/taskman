// Package pipeline drives one task through the full codebase → task →
// commit flow: an isolated worktree, an execution agent, an automated
// format/lint pass, a separate review agent, and a commit agent that writes
// the final message from the actual diff. See RunTask.
package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"uuid"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/taskman/internal/agent"
	"github.com/mhmdkzr/taskman/internal/codebase"
	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/mhmdkzr/taskman/internal/prompts"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/task"
	codebasetools "github.com/mhmdkzr/taskman/internal/tools/codebase"
	"github.com/mhmdkzr/taskman/internal/usage"
)

// Config holds the dependencies and settings RunTask needs, shared across
// every task it runs.
type Config struct {
	Store     *store.Store
	Publisher publisher.Publisher
	Agent     config.AgentConfig

	// SourceDir is the canonical repository RunTask clones from for
	// isolation; each task gets its own clone under WorkDir.
	SourceDir string

	// WorkDir is the parent directory task worktrees are cloned into. Each
	// task gets its own subdirectory, named by its id.
	WorkDir string

	// Author signs the pipeline's commits.
	Author codebase.AuthorSignature

	// MaxFixupRounds bounds how many times the execution agent gets to react
	// to lint findings (before the first commit proposal) or reviewer
	// feedback (after it), per phase, before RunTask gives up on the task.
	// Non-positive uses a default of 3.
	MaxFixupRounds int
}

func (c Config) normalize() Config {
	if c.MaxFixupRounds <= 0 {
		c.MaxFixupRounds = 3
	}
	return c
}

// Result is the outcome of a successful RunTask.
type Result struct {
	TaskID             uuid.UUID
	WorktreeDir        string
	CommitHash         string
	ExecutionSessionID uuid.UUID
	ReviewSessionID    uuid.UUID
}

// executionProposal is the execution agent's structured final answer: it
// believes the task is done and proposes a commit for it.
type executionProposal struct {
	CommitType    string `json:"commit_type"    jsonschema:"description=Conventional-commit type: feat, fix, refactor, chore, test, docs, style, perf, revert, build, ci, or misc."`
	CommitMessage string `json:"commit_message" jsonschema:"description=A one-line, conventional-commit-style summary of the change."`
	Summary       string `json:"summary"        jsonschema:"description=A short note on what was done and why, for the reviewer."`
}

// reviewVerdict is the review agent's structured final answer.
type reviewVerdict struct {
	Approved bool   `json:"approved"           jsonschema:"description=Whether the change passes review as-is."`
	Feedback string `json:"feedback,omitempty" jsonschema:"description=Actionable feedback for the execution agent; required when approved is false."`
}

// commitProposal is the commit agent's structured final answer.
type commitProposal struct {
	Message string `json:"message" jsonschema:"description=The final commit message, formatted per Conventional Commits."`
}

// RunTask drives t through the full pipeline: clone an isolated worktree on
// its own branch, run the execution agent until its work passes the
// automated format/lint pass, run a separate review agent (looping fixups
// back through the execution agent on request-changes), then run a commit
// agent against the actual diff and commit its message. On success the
// worktree is removed; on any error it's left in place for inspection.
func RunTask(ctx context.Context, cfg Config, t task.Task) (*Result, error) {
	cfg = cfg.normalize()

	dir := filepath.Join(cfg.WorkDir, t.ID.String())
	branch := "task/" + t.ID.String()
	repo, err := codebase.Clone(cfg.SourceDir, dir, codebase.CloneOptions{Branch: branch})
	if err != nil {
		return nil, fmt.Errorf("clone worktree: %w", err)
	}

	execOpts, execTools, runner, err := buildExecutorOptions(cfg, dir)
	if err != nil {
		return nil, fmt.Errorf("build executor options: %w", err)
	}

	execSessionID, execSID, err := createSession(ctx, cfg, execOpts, execTools)
	if err != nil {
		return nil, fmt.Errorf("create execution session: %w", err)
	}
	if err := task.StartTask(ctx, cfg.Store.RW(), t.ID, execSID); err != nil {
		return nil, fmt.Errorf("start task: %w", err)
	}

	proposal, execUsage, err := runExecution(ctx, cfg, repo, t, execOpts, execSessionID)
	if err != nil {
		return nil, err
	}
	if runner != nil {
		if err := runner.Wait(ctx); err != nil {
			slog.Warn("pipeline: sub-agents did not finish cleanly", "task_id", t.ID, "error", err)
		}
	}

	if err := task.CompleteTask(
		ctx,
		cfg.Store.RW(),
		t.ID,
		proposal.CommitType,
		proposal.CommitMessage,
		execUsage,
	); err != nil {
		return nil, fmt.Errorf("complete task: %w", err)
	}

	reviewOpts, reviewTools, err := buildReviewerOptions(cfg, repo, dir)
	if err != nil {
		return nil, fmt.Errorf("build reviewer options: %w", err)
	}
	reviewSessionID, reviewSID, err := createSession(ctx, cfg, reviewOpts, reviewTools)
	if err != nil {
		return nil, fmt.Errorf("create review session: %w", err)
	}

	reviewUsage, execFixupUsage, err := runReview(
		ctx,
		cfg,
		repo,
		t,
		execOpts,
		reviewOpts,
		execSessionID,
		reviewSessionID,
	)
	if err != nil {
		return nil, err
	}

	finalMessage, commitAgentUsage, err := runCommitAgent(ctx, cfg, repo, t)
	if err != nil {
		return nil, err
	}

	commitResult, err := repo.Commit(codebase.NewCommitMessage(finalMessage), cfg.Author)
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	totalReviewUsage := reviewUsage.Add(execFixupUsage).Add(commitAgentUsage)
	if err := task.ReviewTask(
		ctx,
		cfg.Store.RW(),
		t.ID,
		reviewSID,
		string(commitResult.Hash),
		totalReviewUsage,
	); err != nil {
		return nil, fmt.Errorf("review task: %w", err)
	}

	if err := repo.Remove(); err != nil {
		slog.Warn("pipeline: failed to remove task worktree", "task_id", t.ID, "dir", dir, "error", err)
	}

	return &Result{
		TaskID:             t.ID,
		WorktreeDir:        dir,
		CommitHash:         string(commitResult.Hash),
		ExecutionSessionID: execSID,
		ReviewSessionID:    reviewSID,
	}, nil
}

// buildExecutorOptions assembles the execution agent's Options and full
// read+write tool set (codebase read/edit/glob/grep/build/test, the task
// backlog, spawn, telegram — see agent.DefaultTools), scoped to dir.
func buildExecutorOptions(cfg Config, dir string) (agent.Options, []goai.Tool, executorRunner, error) {
	sysPrompt, err := prompts.ExecutionAgent()
	if err != nil {
		return agent.Options{}, nil, nil, fmt.Errorf("execution agent prompt: %w", err)
	}
	opts := agent.Options{
		Config:          cfg.Agent,
		Model:           cfg.Agent.Model,
		ReasoningEffort: agent.ReasoningEffort(cfg.Agent.ReasoningEffort),
		MaxSteps:        cfg.Agent.MaxSteps,
		SystemPrompt:    sysPrompt,
		Codebase:        dir,
	}
	tools, runner, err := agent.DefaultTools(cfg.Store, cfg.Publisher, opts)
	if err != nil {
		return agent.Options{}, nil, nil, fmt.Errorf("default tools: %w", err)
	}
	opts.Tools = tools
	return opts, tools, runner, nil
}

// buildReviewerOptions assembles the review agent's Options and read-only
// tool set (no edit, no task backlog, no spawn — see
// codebasetools.ReadOnlyTools), scoped to the same worktree as the executor.
func buildReviewerOptions(cfg Config, repo codebase.Repository, dir string) (agent.Options, []goai.Tool, error) {
	sysPrompt, err := prompts.ReviewAgent()
	if err != nil {
		return agent.Options{}, nil, fmt.Errorf("review agent prompt: %w", err)
	}
	tools := codebasetools.ReadOnlyTools(repo)
	opts := agent.Options{
		Config:          cfg.Agent,
		Model:           cfg.Agent.Model,
		ReasoningEffort: agent.ReasoningEffort(cfg.Agent.ReasoningEffort),
		MaxSteps:        cfg.Agent.MaxSteps,
		SystemPrompt:    sysPrompt,
		Tools:           tools,
		Codebase:        dir,
	}
	return opts, tools, nil
}

// executorRunner is the sub-agent Runner agent.DefaultTools returns; named
// here only so buildExecutorOptions doesn't need to import spawn directly.
type executorRunner = interface {
	Wait(ctx context.Context) error
}

// createSession pre-registers a session row for opts before any turn runs,
// so the caller can reference the session id (e.g. task.StartTask's FK) and
// get an accurate started_at, then run turns against it with
// agent.ContinueSessionObject.
func createSession(
	ctx context.Context,
	cfg Config,
	opts agent.Options,
	tools []goai.Tool,
) (string, uuid.UUID, error) {
	id, err := store.NewSessionID()
	if err != nil {
		return "", uuid.Nil(), err
	}
	if err := store.CreateSession(
		ctx,
		cfg.Store.RW(),
		id,
		opts.Model,
		string(opts.ReasoningEffort),
		opts.MaxSteps,
		opts.SystemPrompt,
		storeTools(tools),
	); err != nil {
		return "", uuid.Nil(), err
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		return "", uuid.Nil(), err
	}
	return id, parsed, nil
}

// runExecution drives the execution agent from the task description until it
// proposes a commit whose result also passes the automated format/lint pass,
// retrying up to cfg.MaxFixupRounds times when lint finds something.
func runExecution(
	ctx context.Context,
	cfg Config,
	repo codebase.Repository,
	t task.Task,
	execOpts agent.Options,
	execSessionID string,
) (executionProposal, usage.TokenUsage, error) {
	var total usage.TokenUsage
	prompt := taskPrompt(t)

	for round := 0; ; round++ {
		proposal, res, _, err := agent.ContinueSessionObject[executionProposal](
			ctx,
			cfg.Store,
			execOpts,
			cfg.Publisher,
			execSessionID,
			prompt,
		)
		if err != nil {
			return executionProposal{}, total, fmt.Errorf("execution agent: %w", err)
		}
		total = total.Add(res.Usage)

		if err := repo.Format(); err != nil {
			return executionProposal{}, total, fmt.Errorf("format: %w", err)
		}
		report, err := scopedLint(repo)
		if err != nil {
			return executionProposal{}, total, err
		}
		if report.Clean() {
			return proposal, total, nil
		}
		if round >= cfg.MaxFixupRounds {
			return executionProposal{}, total, fmt.Errorf("lint still failing after %d round(s):\n%s", round+1, report)
		}
		prompt = fmt.Sprintf(
			"The automated lint gate found issues after your last change. Fix them, then finish again.\n\n%s",
			report,
		)
	}
}

// runReview drives the review agent against the executor's work, looping
// fixups back through the execution session on request-changes, up to
// cfg.MaxFixupRounds rounds. It returns the reviewer's own usage and the
// usage of any post-completion executor fixup turns separately, since both
// get folded into the task's review usage bucket (there's no third bucket
// for "fixups after completion" in the task schema).
func runReview(
	ctx context.Context,
	cfg Config,
	repo codebase.Repository,
	t task.Task,
	execOpts, reviewOpts agent.Options,
	execSessionID, reviewSessionID string,
) (usage.TokenUsage, usage.TokenUsage, error) {
	var reviewUsage, execFixupUsage usage.TokenUsage
	for round := 0; ; round++ {
		diffs, err := repo.Diff()
		if err != nil {
			return reviewUsage, execFixupUsage, fmt.Errorf("diff: %w", err)
		}
		report, err := scopedLint(repo)
		if err != nil {
			return reviewUsage, execFixupUsage, err
		}

		verdict, res, _, err := agent.ContinueSessionObject[reviewVerdict](
			ctx,
			cfg.Store,
			reviewOpts,
			cfg.Publisher,
			reviewSessionID,
			reviewInputPrompt(t, diffs, report),
		)
		if err != nil {
			return reviewUsage, execFixupUsage, fmt.Errorf("review agent: %w", err)
		}
		reviewUsage = reviewUsage.Add(res.Usage)

		if verdict.Approved {
			return reviewUsage, execFixupUsage, nil
		}
		if round >= cfg.MaxFixupRounds {
			return reviewUsage, execFixupUsage, fmt.Errorf(
				"task rejected after %d review round(s): %s",
				round+1,
				verdict.Feedback,
			)
		}

		fixupPrompt := fmt.Sprintf("Review feedback — address it, then finish again.\n\n%s", verdict.Feedback)
		_, execRes, _, err := agent.ContinueSessionObject[executionProposal](
			ctx,
			cfg.Store,
			execOpts,
			cfg.Publisher,
			execSessionID,
			fixupPrompt,
		)
		if err != nil {
			return reviewUsage, execFixupUsage, fmt.Errorf("execution agent fixup: %w", err)
		}
		execFixupUsage = execFixupUsage.Add(execRes.Usage)

		if err := repo.Format(); err != nil {
			return reviewUsage, execFixupUsage, fmt.Errorf("format: %w", err)
		}
	}
}

// runCommitAgent runs the tool-less, single-shot commit agent against the
// actual diff being committed (not the execution agent's self-reported
// summary, which can drift from what the diff really shows) and returns the
// commit message it writes.
func runCommitAgent(
	ctx context.Context,
	cfg Config,
	repo codebase.Repository,
	t task.Task,
) (string, usage.TokenUsage, error) {
	sysPrompt, err := prompts.CommitAgent()
	if err != nil {
		return "", usage.TokenUsage{}, fmt.Errorf("commit agent prompt: %w", err)
	}
	conventions, err := prompts.ConventionalCommits()
	if err != nil {
		return "", usage.TokenUsage{}, fmt.Errorf("conventional commits prompt: %w", err)
	}
	diffs, err := repo.Diff()
	if err != nil {
		return "", usage.TokenUsage{}, fmt.Errorf("diff: %w", err)
	}

	opts := agent.Options{
		Config:          cfg.Agent,
		Model:           cfg.Agent.Model,
		ReasoningEffort: agent.ReasoningEffort(cfg.Agent.ReasoningEffort),
		MaxSteps:        1,
		SystemPrompt:    sysPrompt + "\n\n" + conventions,
	}
	inputPrompt := fmt.Sprintf("# Task\n\n%s\n\n# Diff being committed\n\n%s", taskPrompt(t), formatDiffs(diffs))

	proposal, res, err := agent.RunObject[commitProposal](ctx, opts, cfg.Publisher, inputPrompt, "")
	if err != nil {
		return "", usage.TokenUsage{}, fmt.Errorf("commit agent: %w", err)
	}
	if _, err := agent.PersistRun(ctx, cfg.Store, opts, cfg.Publisher, inputPrompt, res); err != nil {
		return "", usage.TokenUsage{}, fmt.Errorf("persist commit agent run: %w", err)
	}
	return strings.TrimSpace(proposal.Message), res.Usage, nil
}

// storeTools converts an agent's tool set into the shape store.CreateSession
// persists — a plain field-by-field copy, since goai.Tool carries a
// non-serializable Execute function that store.Tool doesn't need.
func storeTools(tools []goai.Tool) []store.Tool {
	out := make([]store.Tool, 0, len(tools))
	for _, t := range tools {
		out = append(out, store.Tool{
			Name:                t.Name,
			Description:         t.Description,
			InputSchema:         string(t.InputSchema),
			ProviderDefinedType: t.ProviderDefinedType,
		})
	}
	return out
}

// taskPrompt renders a task's descriptive spec into the prompt text given to
// the execution and review agents.
func taskPrompt(t task.Task) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", t.Title)
	fmt.Fprintf(
		&b,
		"Type: %s | Urgency: %s | Importance: %s | Risk: %s\n\n",
		t.TaskType,
		t.Urgency,
		t.Importance,
		t.Risk,
	)
	fmt.Fprintf(&b, "## What\n%s\n\n", t.What)
	fmt.Fprintf(&b, "## Why\n%s\n\n", t.Why)
	fmt.Fprintf(&b, "## How\n%s\n\n", t.How)
	if len(t.Invariants) > 0 {
		b.WriteString("## Invariants\n")
		for _, inv := range t.Invariants {
			fmt.Fprintf(&b, "- %s\n", inv)
		}
		b.WriteString("\n")
	}
	if len(t.Where) > 0 {
		b.WriteString("## Where\n")
		for _, w := range t.Where {
			fmt.Fprintf(&b, "- %s\n", w)
		}
		b.WriteString("\n")
	}
	if len(t.CompletedWhen) > 0 {
		b.WriteString("## Completed when\n")
		for _, c := range t.CompletedWhen {
			fmt.Fprintf(&b, "- %s\n", c)
		}
	}
	return b.String()
}

// formatDiffs renders a repository diff into plain text for an agent prompt.
// maxDiffChars bounds formatDiffs' output so an unusually large change
// cannot flood an agent's context window.
const maxDiffChars = 40000

func formatDiffs(diffs []codebase.Diff) string {
	if len(diffs) == 0 {
		return "(no changes)"
	}
	var b strings.Builder
	for _, d := range diffs {
		fmt.Fprintf(&b, "--- %s (%s, +%d/-%d) ---\n%s\n\n", d.Name, d.ChangeType, d.Additions, d.Deletions, d.Patch)
	}
	out := strings.TrimRight(b.String(), "\n")
	if len(out) > maxDiffChars {
		out = out[:maxDiffChars] + fmt.Sprintf("\n... (truncated to %d chars — the diff is larger than shown)", maxDiffChars)
	}
	return out
}

// reviewInputPrompt is the review agent's per-round input: the task spec,
// the current diff, and the automated lint report.
func reviewInputPrompt(t task.Task, diffs []codebase.Diff, lint codebase.LintReport) string {
	return fmt.Sprintf(
		"# Task\n\n%s\n\n# Diff\n\n%s\n\n# Automated lint gate\n\n%s",
		taskPrompt(t),
		formatDiffs(diffs),
		lint.String(),
	)
}
