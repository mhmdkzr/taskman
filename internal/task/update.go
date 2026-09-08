package task

import "maps"

// UpdateRequest is task update's input - patch semantics, only non-nil
// fields are applied. design.md §6.
type UpdateRequest struct {
	Title           *string
	SetLabels       map[string]string
	UnsetLabels     []string
	References      []string
	ClearReferences bool
}

// Update patches a task's metadata (title, labels, references). It never
// touches specification/done_when (own command: Specify) or git/status
// (taskman-managed). No precondition on State/Status - metadata isn't
// workflow state. design.md §6.
func Update(repo *Repo, id string, req UpdateRequest) (Task, error) {
	merged := make(map[string]string)
	return repo.Mutate(id, func(t *Task) error {
		if req.Title != nil {
			t.Title = *req.Title
		}
		if len(req.SetLabels) > 0 || len(req.UnsetLabels) > 0 {
			maps.Copy(merged, t.Labels)
			maps.Copy(merged, req.SetLabels)
			for _, k := range req.UnsetLabels {
				delete(merged, k)
			}
			if err := ValidateLabels(merged); err != nil {
				return err
			}
			t.Labels = merged
		}
		if req.ClearReferences {
			t.References = nil
		} else if len(req.References) > 0 {
			t.References = req.References
		}
		return nil
	})
}
