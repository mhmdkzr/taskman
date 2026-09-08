package task

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

func TestRepoCreateGetList(t *testing.T) {
	repo := NewRepo(filepath.Join(t.TempDir(), "tasks"))

	t1 := Task{ID: "abc", Title: "First", Definition: "def", Labels: map[string]string{"priority": "high"}}
	if err := repo.Create(t1); err != nil {
		t.Fatalf("create: %v", err)
	}
	t2 := Task{ID: "def", Title: "Second", Definition: "def2", Labels: map[string]string{"priority": "low"}}
	if err := repo.Create(t2); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get("abc")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "First" {
		t.Errorf("title = %q, want First", got.Title)
	}

	if _, err := repo.Get("missing"); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("get missing: err = %v, want ErrTaskNotFound", err)
	}

	all, err := repo.List(Filter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("list len = %d, want 2", len(all))
	}
	if all[0].ID != "abc" || all[1].ID != "def" {
		t.Errorf("list order = %v, want [abc def]", []string{all[0].ID, all[1].ID})
	}

	filtered, err := repo.List(Filter{Labels: map[string]string{"priority": "low"}})
	if err != nil {
		t.Fatalf("list filtered: %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != "def" {
		t.Fatalf("list filtered = %+v, want just def", filtered)
	}

	if err := repo.Create(t1); err == nil {
		t.Error("create duplicate id: want error, got nil")
	}
}

func TestRepoMutate(t *testing.T) {
	repo := NewRepo(t.TempDir())
	if err := repo.Create(Task{ID: "abc", Title: "Original", Definition: "def"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Mutate("abc", func(tk *Task) error {
		tk.Title = "Updated"
		return nil
	})
	if err != nil {
		t.Fatalf("mutate: %v", err)
	}
	if got.Title != "Updated" {
		t.Errorf("title = %q, want Updated", got.Title)
	}

	reread, err := repo.Get("abc")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if reread.Title != "Updated" {
		t.Errorf("reread title = %q, want Updated", reread.Title)
	}

	sentinel := errors.New("precondition failed")
	_, err = repo.Mutate("abc", func(tk *Task) error {
		tk.Title = "Should not persist"
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("mutate err = %v, want sentinel", err)
	}
	reread2, err := repo.Get("abc")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if reread2.Title != "Updated" {
		t.Errorf("aborted mutate leaked a write: title = %q, want Updated", reread2.Title)
	}
}

func TestRepoMutateConcurrent(t *testing.T) {
	repo := NewRepo(t.TempDir())
	if err := repo.Create(Task{ID: "abc", Title: "t", Definition: "def", Labels: map[string]string{}}); err != nil {
		t.Fatalf("create: %v", err)
	}

	const n = 50
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := repo.Mutate("abc", func(tk *Task) error {
				if tk.Labels == nil {
					tk.Labels = map[string]string{}
				}
				tk.Labels[keyFor(i)] = "v"
				return nil
			})
			errs[i] = err
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("mutate %d: %v", i, err)
		}
	}

	final, err := repo.Get("abc")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(final.Labels) != n {
		t.Fatalf("labels = %d, want %d - concurrent mutations lost a write", len(final.Labels), n)
	}
}

func keyFor(i int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	return "k" + string(letters[i%len(letters)]) + string(rune('0'+i/len(letters)))
}

func TestRepoDelete(t *testing.T) {
	repo := NewRepo(t.TempDir())
	if err := repo.Create(Task{ID: "abc", Definition: "def"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Delete("abc"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get("abc"); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("get after delete: err = %v, want ErrTaskNotFound", err)
	}
	if err := repo.Delete("abc"); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("delete missing: err = %v, want ErrTaskNotFound", err)
	}
}
