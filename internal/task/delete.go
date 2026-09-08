package task

// Delete removes task id's file outright. No precondition on State/Status -
// git history covers "undo" (design.md §6).
func Delete(repo *Repo, id string) error {
	return repo.Delete(id)
}
