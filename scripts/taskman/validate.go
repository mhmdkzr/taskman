package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

func cmdValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	root := fs.String("root", ".", "repo root that .tasks trees are relative to")
	pkg := fs.String("package", "", "restrict to this package (and its subpackages)")
	fs.Parse(args)

	tasks, warnings, err := walkTasks(*root, *pkg)
	if err != nil {
		die("validate: %v", err)
	}
	problems := append([]string{}, warnings...)

	byID := map[string][]task{}
	for _, t := range tasks {
		byID[t.ID] = append(byID[t.ID], t)
	}
	for id, ts := range byID {
		if len(ts) > 1 {
			paths := make([]string, len(ts))
			for i, t := range ts {
				paths[i] = t.Path
			}
			problems = append(problems, fmt.Sprintf("duplicate id %q: %s", id, strings.Join(paths, ", ")))
		}
	}

	for _, t := range tasks {
		wantPkg, wantSlug, wantID, err := idFromPath(*root, t.Path)
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		if t.ID != wantID {
			problems = append(problems, fmt.Sprintf("%s: id %q does not match its path (want %q)", t.Path, t.ID, wantID))
		}
		if t.Package != wantPkg {
			problems = append(problems, fmt.Sprintf("%s: package %q does not match its path (want %q)", t.Path, t.Package, wantPkg))
		}
		_ = wantSlug
		if err := validateEnum("urgency", t.Urgency, validUrgency); err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", t.Path, err))
		}
		if err := validateEnum("importance", t.Importance, validImportance); err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", t.Path, err))
		}
		if err := validateEnum("type", t.Type, validTypes); err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", t.Path, err))
		}
		if err := validateEnum("status", t.Status, validStatuses); err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", t.Path, err))
		}
		if err := validateEnum("source.kind", t.Source.Kind, validSourceKinds); err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", t.Path, err))
		}
		if t.Status == "open" && (t.Resolved || t.ResolvedAt != nil) {
			problems = append(problems, fmt.Sprintf("%s: status is open but resolved/resolved_at is set", t.Path))
		}
		if t.Status != "open" && (!t.Resolved || t.ResolvedAt == nil) {
			problems = append(problems, fmt.Sprintf("%s: status is %s but resolved is false or resolved_at is unset", t.Path, t.Status))
		}
		if t.Status != "open" && t.Resolution == "" {
			problems = append(problems, fmt.Sprintf("%s: status is %s but has no ## Resolution section", t.Path, t.Status))
		}
		if t.Status != "open" && len(t.Questions) > 0 {
			problems = append(problems, fmt.Sprintf("%s: status is %s but still has %d unanswered question(s)", t.Path, t.Status, len(t.Questions)))
		}
		for _, dep := range t.DependsOn {
			if _, ok := byID[dep]; !ok {
				problems = append(problems, fmt.Sprintf("%s: depends_on %q does not resolve to any task", t.Path, dep))
			}
		}
	}

	if cycle := findCycle(tasks); cycle != nil {
		problems = append(problems, fmt.Sprintf("dependency cycle: %s", strings.Join(cycle, " -> ")))
	}

	for _, p := range problems {
		fmt.Println(p)
	}
	fmt.Printf("%d task(s) checked, %d problem(s)\n", len(tasks), len(problems))
	if len(problems) > 0 {
		os.Exit(1)
	}
}

// findCycle runs a standard three-color DFS over the depends_on graph and
// returns the first cycle found as an ordered slice of ids, or nil if the
// graph is acyclic.
func findCycle(tasks []task) []string {
	edges := map[string][]string{}
	for _, t := range tasks {
		edges[t.ID] = t.DependsOn
	}
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var stack []string
	var cyc []string

	var visit func(id string) bool
	visit = func(id string) bool {
		color[id] = gray
		stack = append(stack, id)
		for _, dep := range edges[id] {
			switch color[dep] {
			case gray:
				// found the back-edge; extract the cycle portion of stack
				start := 0
				for i, s := range stack {
					if s == dep {
						start = i
						break
					}
				}
				cyc = append(append([]string{}, stack[start:]...), dep)
				return true
			case white:
				if visit(dep) {
					return true
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[id] = black
		return false
	}

	ids := make([]string, 0, len(edges))
	for id := range edges {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if color[id] == white {
			if visit(id) {
				return cyc
			}
		}
	}
	return nil
}
