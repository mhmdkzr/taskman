// Package runner executes run commands received on the bus. It is the
// server-side counterpart to the taskman CLI's `run`: the CLI publishes a
// RunCommand, the runner picks it up and drives each task through the
// pipeline, publishing the usual agent.pipeline.* events along the way so the
// read-only dashboard shows the work live.
package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/codebase"
	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/mhmdkzr/taskman/internal/pipeline"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/task"
)

// author signs the pipeline's commits, matching the CLI's historical choice.
var author = codebase.AuthorSignature{
	Name:  "taskman-pipeline",
	Email: "pipeline@taskman.local",
}

// Runner executes RunCommands from the bus. Build it with New and serve it
// with Run.
type Runner struct {
	st    *store.Store
	pub   publisher.Publisher
	agent config.AgentConfig
	work  string // parent dir for task worktrees; created on first use
}

// New builds a Runner over the shared store and publisher.
func New(st *store.Store, pub publisher.Publisher, agentCfg config.AgentConfig) *Runner {
	return &Runner{st: st, pub: pub, agent: agentCfg}
}

// Run subscribes to the run-command subject and serves commands until ctx is
// cancelled. Each RunCommand's tasks run concurrently, each in its own
// worktree.
func (r *Runner) Run(ctx context.Context) error {
	stop, err := r.pub.SubscribeCore(events.CommandRunSubject, func(_ string, data []byte) {
		var cmd events.RunCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			slog.Error("runner: bad run command", "error", err)
			return
		}
		for _, id := range cmd.TaskIDs {
			go r.runTask(ctx, cmd, id)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe run commands: %w", err)
	}
	defer stop()
	<-ctx.Done()
	return nil
}

// runTask executes one task id from a command in its own goroutine and
// worktree.
func (r *Runner) runTask(ctx context.Context, cmd events.RunCommand, rawID string) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		slog.Error("runner: invalid task id in command", "id", rawID, "error", err)
		return
	}
	t, err := task.GetTask(ctx, r.st.RW(), id)
	if err != nil {
		slog.Error("runner: get task for command", "task_id", id, "error", err)
		return
	}
	work, err := r.workDir()
	if err != nil {
		slog.Error("runner: workdir", "task_id", id, "error", err)
		return
	}
	cfg := pipeline.Config{
		Store:     r.st,
		Publisher: r.pub,
		Agent:     r.agent,
		SourceDir: cmd.Source,
		WorkDir:   work,
		Author:    author,
	}
	if cfg.SourceDir == "" {
		cfg.SourceDir = "."
	}
	if abs, err := filepath.Abs(cfg.SourceDir); err == nil {
		cfg.SourceDir = abs
	}

	start := time.Now()
	if _, err := pipeline.RunTask(ctx, cfg, *t); err != nil {
		slog.Error("runner: command run failed", "task_id", id, "error", err, "elapsed", time.Since(start))
		return
	}
	slog.Info("runner: command run finished", "task_id", id, "elapsed", time.Since(start))
}

// workDir lazily creates the parent directory task worktrees are cloned into.
func (r *Runner) workDir() (string, error) {
	if r.work != "" {
		return r.work, nil
	}
	work, err := os.MkdirTemp("", "taskman-run-")
	if err != nil {
		return "", fmt.Errorf("create work dir: %w", err)
	}
	r.work = work
	return work, nil
}
