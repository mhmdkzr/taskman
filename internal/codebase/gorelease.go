package codebase

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

type GoReleaseArgs struct {
	Base    string
	Version string
}

type GoReleaseResult struct {
	Packages    []GoReleasePackage
	Diagnostics []string
	Summary     GoReleaseSummary
}

type GoReleasePackage struct {
	Path                string
	CompatibleChanges   []string
	IncompatibleChanges []string
	BaseErrors          []string
	ReleaseErrors       []string
}

type GoReleaseSummary struct {
	BaseVersion      string
	Version          string
	Valid            bool
	SuggestedVersion string
	InvalidReason    string
	Warnings         []string
}

func (r Repository) GoRelease(args GoReleaseArgs) (*GoReleaseResult, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	cmdArgs := []string{"run", "golang.org/x/exp/cmd/gorelease@latest"}
	if args.Base != "" {
		cmdArgs = append(cmdArgs, "-base", args.Base)
	}
	if args.Version != "" {
		cmdArgs = append(cmdArgs, "-version", args.Version)
	}

	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = wt.Filesystem.Root()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("gorelease: %w", err)
	}

	res := &GoReleaseResult{}
	var currentPkg *GoReleasePackage
	var subsection string

	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := sc.Text()

		if strings.HasPrefix(line, "# ") && !strings.HasPrefix(line, "## ") {
			section := line[2:]
			switch section {
			case "diagnostics":
				currentPkg = nil
				subsection = ""
			case "summary":
				currentPkg = nil
				subsection = "summary"
			default:
				res.Packages = append(res.Packages, GoReleasePackage{Path: section})
				currentPkg = &res.Packages[len(res.Packages)-1]
				subsection = ""
			}
			continue
		}

		if strings.HasPrefix(line, "## ") {
			subsection = line[3:]
			continue
		}

		if line == "" {
			continue
		}

		text := strings.TrimPrefix(line, "- ")

		if subsection == "summary" {
			res.parseSummaryLine(text)
			continue
		}

		if currentPkg == nil {
			res.Diagnostics = append(res.Diagnostics, text)
			continue
		}

		switch subsection {
		case "compatible changes":
			currentPkg.CompatibleChanges = append(currentPkg.CompatibleChanges, text)
		case "incompatible changes":
			currentPkg.IncompatibleChanges = append(currentPkg.IncompatibleChanges, text)
		case "errors in base version:":
			currentPkg.BaseErrors = append(currentPkg.BaseErrors, text)
		case "errors in release version:":
			currentPkg.ReleaseErrors = append(currentPkg.ReleaseErrors, text)
		}
	}

	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read gorelease output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return res, fmt.Errorf("gorelease: %w", err)
	}

	return res, nil
}

func (r *GoReleaseResult) parseSummaryLine(text string) {
	switch {
	case strings.HasPrefix(text, "Base version:"):
		r.Summary.BaseVersion = strings.TrimSpace(text[len("Base version:"):])
	case strings.HasPrefix(text, "Inferred base version:"):
		r.Summary.BaseVersion = strings.TrimSpace(text[len("Inferred base version:"):])
	case strings.HasPrefix(text, "Suggested version:"):
		r.Summary.SuggestedVersion = strings.TrimSpace(text[len("Suggested version:"):])
	case strings.HasPrefix(text, "is a valid semantic version for this release"):
		r.Summary.Valid = true
	case strings.HasPrefix(text, "is not a valid semantic version for this release"):
		r.Summary.Valid = false
		r.Summary.InvalidReason = text
	default:
		r.Summary.Warnings = append(r.Summary.Warnings, text)
	}
}
