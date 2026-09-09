package task

import "fmt"

// ValidateLabels checks the well-known label keys (priority, complexity,
// autonomy) against their fixed low/medium/high enum when present. Every
// other key is an unchecked plain user tag.
func ValidateLabels(labels map[string]string) error {
	for _, key := range []string{LabelPriority, LabelComplexity, LabelAutonomy} {
		value, ok := labels[key]
		if !ok {
			continue
		}
		switch value {
		case LevelLow, LevelMedium, LevelHigh:
		default:
			return fmt.Errorf("%w: %s must be %q, %q, or %q, got %q",
				ErrInvalidLabel, key, LevelLow, LevelMedium, LevelHigh, value)
		}
	}
	return nil
}
