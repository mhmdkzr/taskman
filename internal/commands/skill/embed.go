// Package skill owns the "skill" command: it embeds this repo's own
// agent-facing driver document (SKILL.md) into the taskman binary, so
// `taskman skill` can print it without a checkout of this repo on hand.
package skill

import _ "embed"

//go:embed SKILL.md
var content string

// Content returns the embedded SKILL.md text, exactly as written.
func Content() string {
	return content
}
