package main

import (
	"fmt"
	"slices"
	"strings"
)

// ---------------------------------------------------------------------
// Frontmatter schema
// ---------------------------------------------------------------------

// Severity is deliberately not a field: it conflated two independent axes.
// urgency = how soon this needs attention (time pressure); importance = how
// much it matters if never done (impact). Three levels each, orthogonal.
var validUrgency = []string{"low", "medium", "high"}
var validImportance = []string{"low", "medium", "high"}
var validTypes = []string{"bug-fix", "test-gap", "doc-drift", "convention", "feature", "operational"}
var validStatuses = []string{"open", "done", "dropped"}
var validSourceKinds = []string{"review", "session", "user-request"}

type source struct {
	Kind string `yaml:"kind"`
	Ref  string `yaml:"ref"`
}

type frontmatter struct {
	ID         string   `yaml:"id"`
	Package    string   `yaml:"package"`
	Title      string   `yaml:"title"`
	Status     string   `yaml:"status"`
	Urgency    string   `yaml:"urgency"`
	Importance string   `yaml:"importance"`
	Type       string   `yaml:"type"`
	Tags       []string `yaml:"tags"`
	DependsOn  []string `yaml:"depends_on"`
	Questions  []string `yaml:"questions"`
	Where      []string `yaml:"where"`
	Source     source   `yaml:"source"`
	Generated  string   `yaml:"generated"`
	Resolved   bool     `yaml:"resolved"`
	ResolvedAt *string  `yaml:"resolved_at,omitempty"`
}

// task is a fully parsed task file: frontmatter plus its body sections.
type task struct {
	Path string
	frontmatter
	What       string
	How        string
	Why        string
	DoneWhen   string
	Resolution string
}

func validateEnum(field, value string, allowed []string) error {
	if !slices.Contains(allowed, value) {
		return fmt.Errorf("%s: %q is not one of %s", field, value, strings.Join(allowed, ", "))
	}
	return nil
}
