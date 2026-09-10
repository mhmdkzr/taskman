package taskv1

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v3"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

type Change struct {
	ID   string     `json:"id"`
	From string     `json:"from"`
	To   task.State `json:"to"`
}

type Result struct {
	Migrated []Change `json:"migrated,omitempty"`
	Skipped  []string `json:"skipped,omitempty"`
}

type candidate struct {
	path     string
	original []byte
	task     task.Task
	change   Change
}

// Migrate validates every task file before atomically rewriting any of them.
func Migrate(tasksDir string, dryRun bool) (Result, error) {
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, nil
		}
		return Result{}, fmt.Errorf("read tasks dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	result := Result{}
	var candidates []candidate
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(tasksDir, entry.Name())
		// #nosec G304 -- entry.Name comes directly from os.ReadDir(tasksDir).
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		var doc document
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return Result{}, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		var current currentDocument
		if err := yaml.Unmarshal(data, &current); err != nil {
			return Result{}, fmt.Errorf("parse current task %s: %w", entry.Name(), err)
		}
		if !doc.Task.Status.present() && task.Validate(current.Task) == nil {
			result.Skipped = append(result.Skipped, current.Task.ID)
			continue
		}
		converted, err := convert(doc.Task)
		if err != nil {
			return Result{}, fmt.Errorf("migrate task %s: %w", doc.Task.ID, err)
		}
		change := Change{ID: converted.ID, From: doc.Task.State, To: converted.State}
		result.Migrated = append(result.Migrated, change)
		candidates = append(candidates, candidate{path: path, original: data, task: converted, change: change})
	}
	if dryRun {
		return result, nil
	}

	for _, item := range candidates {
		lock := flock.New(item.path + ".lock")
		if err := lock.Lock(); err != nil {
			return Result{}, fmt.Errorf("lock task %s: %w", item.change.ID, err)
		}
		current, err := os.ReadFile(item.path)
		if err == nil && !bytes.Equal(current, item.original) {
			err = fmt.Errorf("task changed during migration")
		}
		if err == nil {
			err = store.Write(tasksDir, item.task)
		}
		unlockErr := lock.Unlock()
		if err != nil {
			return Result{}, fmt.Errorf("migrate task %s: %w", item.change.ID, err)
		}
		if unlockErr != nil {
			return Result{}, fmt.Errorf("unlock task %s: %w", item.change.ID, unlockErr)
		}
	}
	return result, nil
}
