// Package prune owns the "prune" command: its domain logic and CLI wiring.
// It deletes the task files of every completed task, so completed tasks can
// be cleared out in one shot rather than one `delete <id>` at a time.
package prune

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is prune's input. Shared verbatim by the CLI (cmd.go builds it
// from the --dry-run flag) and MCP (mcp.go uses it as the tool's input type
// directly) frontends; the jsonschema tag describes it to MCP clients.
type Request struct {
	// DryRun lists the completed tasks that would be deleted without
	// actually removing anything.
	DryRun bool `json:"dry_run,omitempty" jsonschema:"list completed tasks without deleting them"`
}

// Result is Prune's output: the completed task ids that were pruned (or, in
// dry-run mode, would be pruned) and how many there were.
type Result struct {
	Deleted []string `json:"deleted,omitempty"`
	Count   int      `json:"count"`
}

// Prune removes the task files of every completed task under tasksDir. It
// never touches anything but .tasks/*.yaml - no soft-delete and no git
// worktree/branch cleanup; git history covers "undo", matching task delete.
// With req.DryRun it reports the ids without removing them.
func Prune(tasksDir string, req Request) (Result, error) {
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, nil
		}
		return Result{}, fmt.Errorf("read tasks dir: %w", err)
	}

	var ids []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(entry.Name(), ".yaml"))
	}
	sort.Strings(ids)

	var pruned []string
	for _, id := range ids {
		t, err := store.Read(tasksDir, id)
		if err != nil {
			return Result{}, fmt.Errorf("read task: %w", err)
		}
		if t.State != task.StateCompleted {
			continue
		}
		if !req.DryRun {
			if err := os.Remove(filepath.Join(tasksDir, id+".yaml")); err != nil {
				return Result{}, fmt.Errorf("delete task %s: %w", id, err)
			}
		}
		pruned = append(pruned, id)
	}

	return Result{
		Deleted: pruned,
		Count:   len(pruned),
	}, nil
}
