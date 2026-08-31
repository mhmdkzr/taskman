package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func cmdSetStatus(args []string, status, noteFlag string) {
	fs := flag.NewFlagSet(status, flag.ExitOnError)
	root := fs.String("root", ".", "repo root that .tasks trees are relative to")
	note := fs.String(noteFlag, "", "explanation, appended as a ## Resolution section (required, or -file variant)")
	noteFile := fs.String(noteFlag+"-file", "", "read the "+noteFlag+" text from file")
	fs.Parse(args)
	if fs.NArg() != 1 {
		die("%s: expected exactly one task id", status)
	}
	noteText, err := textOrFile(*note, *noteFile)
	if err != nil {
		die("%s: --%s-file: %v", status, noteFlag, err)
	}
	if strings.TrimSpace(noteText) == "" {
		die("%s: --%s (or --%s-file) is required", status, noteFlag, noteFlag)
	}

	t, err := findTask(*root, fs.Arg(0))
	if err != nil {
		die("%s: %v", status, err)
	}
	if t.Status != "open" {
		die("%s: %s is already %s", status, t.ID, t.Status)
	}
	t.Status = status
	t.Resolved = true
	resolvedAt := time.Now().Format(time.RFC3339)
	t.ResolvedAt = &resolvedAt
	t.Resolution = strings.TrimSpace(noteText)

	out, err := renderTask(t.frontmatter, t.What, t.How, t.Why, t.DoneWhen)
	if err != nil {
		die("%s: %v", status, err)
	}
	out += "\n## Resolution\n" + t.Resolution + "\n"
	if err := os.WriteFile(t.Path, []byte(out), 0o644); err != nil {
		die("%s: %v", status, err)
	}
	fmt.Println(t.ID)
}
