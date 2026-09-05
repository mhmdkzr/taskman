package patch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var errPatchRequired = errors.New("patchText is required")

type hunk struct {
	kind     string
	path     string
	movePath string
	contents string
	chunks   []chunk
}

type chunk struct {
	oldLines []string
	newLines []string
	context  string
	eof      bool
}

type change struct {
	hunk   hunk
	path   string
	old    string
	new    string
	mode   os.FileMode
	oldDir bool
}

func execute(_ context.Context, in input) (output, error) {
	hunks, err := parse(in.PatchText)
	if err != nil {
		return output{}, fmt.Errorf("apply_patch: %w", err)
	}
	if len(hunks) == 0 {
		return output{}, fmt.Errorf("apply_patch: no hunks found")
	}

	changes := make([]change, 0, len(hunks))
	for _, hunk := range hunks {
		prepared, err := prepare(hunk)
		if err != nil {
			return output{}, fmt.Errorf("apply_patch: %w", err)
		}
		changes = append(changes, prepared)
	}

	var result output
	for _, item := range changes {
		if err := apply(item); err != nil {
			return output{}, fmt.Errorf("apply_patch: %w", err)
		}
		switch item.hunk.kind {
		case "add":
			result.Added = append(result.Added, item.path)
		case "delete":
			result.Deleted = append(result.Deleted, item.path)
		default:
			path := item.path
			if item.hunk.movePath != "" {
				path = absolute(item.hunk.movePath)
			}
			result.Modified = append(result.Modified, path)
		}
	}
	return result, nil
}

func parse(text string) ([]hunk, error) {
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n"), "\r", "\n"), "\n")
	begin, end := -1, -1
	for i, line := range lines {
		switch strings.TrimSpace(line) {
		case "*** Begin Patch":
			begin = i
		case "*** End Patch":
			end = i
		}
	}
	if begin < 0 || end <= begin {
		return nil, fmt.Errorf("invalid patch format: missing begin/end markers")
	}

	var hunks []hunk
	for i := begin + 1; i < end; {
		line := lines[i]
		kind, path := header(line, "*** Add File:")
		if kind != "" {
			contents, next := addContents(lines, i+1, end)
			hunks = append(hunks, hunk{kind: "add", path: path, contents: contents})
			i = next
			continue
		}
		kind, path = header(line, "*** Delete File:")
		if kind != "" {
			hunks = append(hunks, hunk{kind: "delete", path: path})
			i++
			continue
		}
		kind, path = header(line, "*** Update File:")
		if kind == "" {
			i++
			continue
		}
		movePath := ""
		i++
		if i < end {
			_, movePath = header(lines[i], "*** Move to:")
			if movePath != "" {
				i++
			}
		}
		chunks, next, err := updateChunks(lines, i, end)
		if err != nil {
			return nil, err
		}
		hunks = append(hunks, hunk{kind: "update", path: path, movePath: movePath, chunks: chunks})
		i = next
	}
	return hunks, nil
}

func header(line, prefix string) (string, string) {
	if !strings.HasPrefix(line, prefix) {
		return "", ""
	}
	path := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if path == "" {
		return "", ""
	}
	return prefix, path
}

func addContents(lines []string, start, end int) (string, int) {
	var contents strings.Builder
	for i := start; i < end && !strings.HasPrefix(lines[i], "***"); i++ {
		if strings.HasPrefix(lines[i], "+") {
			contents.WriteString(strings.TrimPrefix(lines[i], "+"))
			contents.WriteByte('\n')
		}
	}
	return strings.TrimSuffix(contents.String(), "\n"), start + strings.Count(strings.Join(lines[start:end], "\n"), "\n") + 1
}

func updateChunks(lines []string, start, end int) ([]chunk, int, error) {
	var chunks []chunk
	i := start
	for i < end && !strings.HasPrefix(lines[i], "***") {
		if !strings.HasPrefix(lines[i], "@@") {
			i++
			continue
		}
		current := chunk{context: strings.TrimSpace(strings.TrimPrefix(lines[i], "@@"))}
		i++
		for i < end && !strings.HasPrefix(lines[i], "@@") && !strings.HasPrefix(lines[i], "***") {
			line := lines[i]
			switch {
			case line == "*** End of File":
				current.eof = true
			case strings.HasPrefix(line, " "):
				value := strings.TrimPrefix(line, " ")
				current.oldLines = append(current.oldLines, value)
				current.newLines = append(current.newLines, value)
			case strings.HasPrefix(line, "-"):
				current.oldLines = append(current.oldLines, strings.TrimPrefix(line, "-"))
			case strings.HasPrefix(line, "+"):
				current.newLines = append(current.newLines, strings.TrimPrefix(line, "+"))
			default:
				return nil, i, fmt.Errorf("invalid update line %q", line)
			}
			i++
		}
		chunks = append(chunks, current)
	}
	return chunks, i, nil
}

func prepare(hunk hunk) (change, error) {
	path := absolute(hunk.path)
	switch hunk.kind {
	case "add":
		if _, err := os.Stat(path); err == nil {
			return change{}, fmt.Errorf("file already exists: %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return change{}, fmt.Errorf("stat %s: %w", path, err)
		}
		return change{hunk: hunk, path: path, new: ensureNewline(hunk.contents), mode: 0o644}, nil
	case "delete":
		info, err := os.Stat(path)
		if err != nil {
			return change{}, fmt.Errorf("read %s: %w", path, err)
		}
		if info.IsDir() {
			return change{}, fmt.Errorf("path is a directory: %s", path)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return change{}, fmt.Errorf("read %s: %w", path, err)
		}
		return change{hunk: hunk, path: path, old: string(content), mode: info.Mode()}, nil
	case "update":
		info, err := os.Stat(path)
		if err != nil {
			return change{}, fmt.Errorf("read %s: %w", path, err)
		}
		if info.IsDir() {
			return change{}, fmt.Errorf("path is a directory: %s", path)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return change{}, fmt.Errorf("read %s: %w", path, err)
		}
		updated, err := applyChunks(string(content), hunk.chunks, path)
		if err != nil {
			return change{}, err
		}
		return change{hunk: hunk, path: path, old: string(content), new: updated, mode: info.Mode()}, nil
	default:
		return change{}, fmt.Errorf("unknown patch operation %q", hunk.kind)
	}
}

func applyChunks(content string, chunks []chunk, path string) (string, error) {
	trailingNewline := strings.HasSuffix(content, "\n")
	lines := strings.Split(strings.TrimSuffix(strings.ReplaceAll(content, "\r\n", "\n"), "\n"), "\n")
	if content == "" {
		lines = nil
	}
	for _, current := range chunks {
		start := findLines(lines, current.oldLines, current.eof)
		if start < 0 {
			return "", fmt.Errorf("failed to find expected lines in %s:\n%s", path, strings.Join(current.oldLines, "\n"))
		}
		lines = append(append(append([]string{}, lines[:start]...), current.newLines...), lines[start+len(current.oldLines):]...)
	}
	updated := strings.Join(lines, "\n")
	if trailingNewline || updated != "" {
		updated += "\n"
	}
	return updated, nil
}

func findLines(lines, pattern []string, eof bool) int {
	if len(pattern) == 0 {
		if eof {
			return len(lines)
		}
		return 0
	}
	start := 0
	if eof {
		start = len(lines) - len(pattern)
	}
	for i := start; i <= len(lines)-len(pattern); i++ {
		if matchLines(lines[i:i+len(pattern)], pattern) {
			return i
		}
	}
	return -1
}

func matchLines(actual, expected []string) bool {
	for i := range actual {
		if actual[i] == expected[i] || strings.TrimRight(actual[i], " \t") == strings.TrimRight(expected[i], " \t") || strings.TrimSpace(actual[i]) == strings.TrimSpace(expected[i]) {
			continue
		}
		return false
	}
	return true
}

func apply(item change) error {
	switch item.hunk.kind {
	case "add", "update":
		target := item.path
		if item.hunk.movePath != "" {
			target = absolute(item.hunk.movePath)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create parent directory: %w", err)
		}
		if err := os.WriteFile(target, []byte(item.new), item.mode); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
		if item.hunk.movePath != "" {
			if err := os.Remove(item.path); err != nil {
				return fmt.Errorf("remove moved file %s: %w", item.path, err)
			}
		}
	case "delete":
		if err := os.Remove(item.path); err != nil {
			return fmt.Errorf("remove %s: %w", item.path, err)
		}
	}
	return nil
}

func absolute(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return absolutePath
}

func ensureNewline(content string) string {
	if content == "" || strings.HasSuffix(content, "\n") {
		return content
	}
	return content + "\n"
}
