package task

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v3"
)

const filePerm = 0o600

// document is the on-disk shape: a task file is { task: <Task> }, matching
// design.md §3's schema example.
type document struct {
	Task Task `yaml:"task"`
}

// Repo is the file-backed task repository - design.md §3's "Repository
// implementation". It is safe for concurrent use, including from separate
// processes: every mutation takes an OS-level advisory lock on the task's
// own file for the duration of its read-modify-write-rename cycle.
type Repo struct {
	dir string
}

// NewRepo returns a Repo rooted at dir (design.md §1's --tasks-dir).
func NewRepo(dir string) *Repo {
	return &Repo{dir: dir}
}

// Get reads the current state of task id.
func (r *Repo) Get(id string) (Task, error) {
	return r.read(id)
}

// Filter narrows List's results. A zero Filter matches every task.
type Filter struct {
	State  []State
	Labels map[string]string
}

func (f Filter) matches(t Task) bool {
	if len(f.State) > 0 {
		found := slices.Contains(f.State, t.State)
		if !found {
			return false
		}
	}
	for k, v := range f.Labels {
		if t.Labels[k] != v {
			return false
		}
	}
	return true
}

// List returns every task matching filter, ordered by id.
func (r *Repo) List(filter Filter) ([]Task, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	var ids []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(entry.Name(), ".yaml"))
	}
	sort.Strings(ids)

	tasks := make([]Task, 0, len(ids))
	for _, id := range ids {
		t, err := r.read(id)
		if err != nil {
			return nil, fmt.Errorf("read task %s: %w", id, err)
		}
		if filter.matches(t) {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}

// Create writes a brand-new task file. It fails if a file for t.ID already
// exists - task ids are <uuid-v7>_<slug> (§3), so this should only ever
// happen if a caller passes in an id by hand.
func (r *Repo) Create(t Task) error {
	if t.ID == "" {
		return fmt.Errorf("create task: id is required")
	}
	if err := os.MkdirAll(r.dir, 0o700); err != nil {
		return fmt.Errorf("create tasks dir: %w", err)
	}
	path := r.path(t.ID)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("create task: %s already exists", t.ID)
	}
	return r.write(t)
}

// Mutate locks task id's file, loads it, calls fn to validate and apply a
// transition, and writes the result back - the read-modify-write-rename
// cycle every command in design.md §6 goes through. fn returning an error
// aborts the mutation: nothing is written, and that error is returned as
// Mutate's own.
func (r *Repo) Mutate(id string, fn func(t *Task) error) (Task, error) {
	// The lock is taken on a separate, stable file - never on the task
	// file itself. write()'s atomic rename replaces the task file's inode
	// on every write; a lock held on a path survives only as long as the
	// inode it was opened against, so locking the renamed file directly
	// would let a concurrent process's fresh open() onto the new inode
	// acquire an independent, non-contending lock - silently defeating
	// cross-process exclusion the moment a write ever succeeded.
	lock := flock.New(r.path(id) + ".lock")
	if err := lock.Lock(); err != nil {
		return Task{}, fmt.Errorf("lock task %s: %w", id, err)
	}
	defer func() {
		if err := lock.Unlock(); err != nil {
			slog.Error("unlock task file", "id", id, "error", err)
		}
	}()

	t, err := r.read(id)
	if err != nil {
		return Task{}, err
	}
	if err := fn(&t); err != nil {
		return Task{}, err
	}
	if err := r.write(t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// Delete removes task id's file outright - no soft-delete, git history
// covers "undo" (§3).
func (r *Repo) Delete(id string) error {
	path := r.path(id)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("stat task %s: %w", id, err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete task %s: %w", id, err)
	}
	return nil
}

func (r *Repo) path(id string) string {
	return filepath.Join(r.dir, id+".yaml")
}

func (r *Repo) read(id string) (Task, error) {
	data, err := os.ReadFile(r.path(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Task{}, ErrTaskNotFound
		}
		return Task{}, fmt.Errorf("read task %s: %w", id, err)
	}
	var doc document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return Task{}, fmt.Errorf("parse task %s: %w", id, err)
	}
	return doc.Task, nil
}

func (r *Repo) write(t Task) error {
	data, err := yaml.Marshal(document{Task: t})
	if err != nil {
		return fmt.Errorf("marshal task %s: %w", t.ID, err)
	}
	path := r.path(t.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, filePerm); err != nil {
		return fmt.Errorf("write task %s: %w", t.ID, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename task %s: %w", t.ID, err)
	}
	return nil
}

// now is a seam for tests; production code always uses time.Now().
var now = func() time.Time { return time.Now().UTC() }
