package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------
// Parsing (list/show/done/drop/validate)
// ---------------------------------------------------------------------

var sectionHeader = regexp.MustCompile(`^##\s+(.+?)\s*$`)

func parseTaskFile(path string) (task, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return task{}, err
	}
	content := string(b)
	if !strings.HasPrefix(content, "---\n") {
		return task{}, fmt.Errorf("%s: missing frontmatter delimiter", path)
	}
	rest := content[len("---\n"):]
	before, after, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return task{}, fmt.Errorf("%s: unterminated frontmatter block", path)
	}
	fmBlock := before
	body := after

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(fmBlock), &fm); err != nil {
		return task{}, fmt.Errorf("%s: parse frontmatter: %w", path, err)
	}

	t := task{Path: path, frontmatter: fm}
	sections := map[string]*strings.Builder{}
	var current *strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if m := sectionHeader.FindStringSubmatch(line); m != nil {
			name := strings.ToLower(m[1])
			b := &strings.Builder{}
			sections[name] = b
			current = b
			continue
		}
		if strings.HasPrefix(line, "# ") || current == nil {
			continue // title heading or preamble before the first "## " section
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return task{}, fmt.Errorf("%s: read body: %w", path, err)
	}
	get := func(name string) string {
		if b, ok := sections[name]; ok {
			return strings.TrimSpace(b.String())
		}
		return ""
	}
	t.What = get("what")
	t.How = get("how")
	t.Why = get("why")
	t.DoneWhen = get("done when")
	t.Resolution = get("resolution")
	return t, nil
}

// walkTasks finds every task file under root, optionally restricted to a
// package subtree (pkgFilter as a path prefix). Parse failures are
// collected as warnings rather than aborting the whole walk, since list/
// search should degrade gracefully in front of a stray malformed file.
func walkTasks(root, pkgFilter string) (tasks []task, warnings []string, err error) {
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() || d.Name() != ".tasks" {
			return nil
		}
		if pkgFilter != "" {
			rel, relErr := filepath.Rel(root, filepath.Dir(path))
			if relErr != nil {
				return relErr
			}
			pkg := filepath.ToSlash(rel)
			if pkg != pkgFilter && !strings.HasPrefix(pkg, pkgFilter+"/") {
				return nil
			}
		}
		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", path, readErr))
			return nil
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			full := filepath.Join(path, e.Name())
			t, perr := parseTaskFile(full)
			if perr != nil {
				warnings = append(warnings, perr.Error())
				continue
			}
			tasks = append(tasks, t)
		}
		return nil
	})
	if pkgFilter != "" {
		filtered := tasks[:0]
		for _, t := range tasks {
			if t.Package == pkgFilter || strings.HasPrefix(t.Package, pkgFilter+"/") {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks, warnings, err
}

func findTask(root, id string) (task, error) {
	tasks, _, err := walkTasks(root, "")
	if err != nil {
		return task{}, err
	}
	for _, t := range tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return task{}, fmt.Errorf("no task with id %q found under %s", id, root)
}
