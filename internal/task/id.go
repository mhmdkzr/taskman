package task

import (
	"strings"
	"unicode"
	"uuid"
)

// GenerateID returns a new <uuid-v7>_<slug> task id. The
// slug comes from title; a title with no letters or digits at all falls
// back to "task".
func GenerateID(title string) string {
	return uuid.NewV7().String() + "_" + Slugify(title)
}

// Slugify lowercases title, replaces runs of non-alphanumeric characters
// with a single hyphen, and trims leading/trailing hyphens. A title with no
// letters or digits at all falls back to "task".
func Slugify(title string) string {
	var b strings.Builder
	lastHyphen := true // avoid a leading hyphen
	for _, r := range title {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			lastHyphen = false
		default:
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	slug := strings.TrimRight(b.String(), "-")
	if slug == "" {
		return "task"
	}
	return slug
}
