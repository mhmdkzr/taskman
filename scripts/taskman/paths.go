package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------
// Slugs and paths
// ---------------------------------------------------------------------

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(title string) string {
	s := strings.ToLower(title)
	s = slugNonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = strings.TrimRight(s[:60], "-")
	}
	if s == "" {
		s = "task"
	}
	return s
}

func tasksDir(root, pkg string) string {
	return filepath.Join(root, pkg, ".tasks")
}

func taskPath(root, pkg, slug string) string {
	return filepath.Join(tasksDir(root, pkg), slug+".md")
}

func idFromPath(root, path string) (pkg, slug, id string, err error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", "", "", err
	}
	rel = filepath.ToSlash(rel)
	dir, file := filepath_SplitSlash(rel)
	if filepath_Base(dir) != ".tasks" {
		return "", "", "", fmt.Errorf("%s: not under a .tasks directory", path)
	}
	pkg = strings.TrimSuffix(dir, "/.tasks")
	slug = strings.TrimSuffix(file, ".md")
	return pkg, slug, pkg + "/" + slug, nil
}

// tiny slash-path helpers so idFromPath doesn't depend on OS path semantics
// once rel has already been normalized to "/".
func filepath_SplitSlash(p string) (dir, file string) {
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return "", p
	}
	return p[:i], p[i+1:]
}

func filepath_Base(p string) string {
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return p
	}
	return p[i+1:]
}
