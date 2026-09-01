package task

import (
	"fmt"
	"slices"
)

// These mirror internal/task's unexported valid-value lists (task.Task.Validate
// checks the same sets, but also requires CommitType/session/timestamp fields
// that aren't known yet at tool-call time, so the tools validate their own
// input shape here rather than constructing a throwaway Task just to call it).
var (
	validTaskTypes = []string{"bug", "docs", "feature", "refactor", "test"}
	validLevels    = []string{"low", "medium", "high"}
	validVariants  = []string{"low", "medium", "high", "xhigh", "max", "default"}
)

// validateSpec checks the descriptive-spec fields shared by task_create and
// task_edit: required strings are non-empty, enums are one of their known
// values, and the two list fields are non-nil (an agent must state at least
// an empty list on purpose, not omit the field). It does not check Variant —
// only task_create sets that, so it's validated separately via validateVariant.
func validateSpec(
	taskType, urgency, importance, risk, title, what, why, how string,
	packages, completedWhen []string,
) error {
	if !slices.Contains(validTaskTypes, taskType) {
		return fmt.Errorf("invalid task_type %q: want one of %v", taskType, validTaskTypes)
	}
	if !slices.Contains(validLevels, urgency) {
		return fmt.Errorf("invalid urgency %q: want one of %v", urgency, validLevels)
	}
	if !slices.Contains(validLevels, importance) {
		return fmt.Errorf("invalid importance %q: want one of %v", importance, validLevels)
	}
	if !slices.Contains(validLevels, risk) {
		return fmt.Errorf("invalid risk %q: want one of %v", risk, validLevels)
	}
	if title == "" {
		return fmt.Errorf("title is required")
	}
	if what == "" {
		return fmt.Errorf("what is required")
	}
	if why == "" {
		return fmt.Errorf("why is required")
	}
	if how == "" {
		return fmt.Errorf("how is required")
	}
	if packages == nil {
		return fmt.Errorf("packages is required (pass an empty list if none apply)")
	}
	if completedWhen == nil {
		return fmt.Errorf("completed_when is required (pass an empty list if none apply)")
	}
	return nil
}

// validateVariant checks a reasoning-effort variant against its known values.
func validateVariant(variant string) error {
	if !slices.Contains(validVariants, variant) {
		return fmt.Errorf("invalid variant %q: want one of %v", variant, validVariants)
	}
	return nil
}
