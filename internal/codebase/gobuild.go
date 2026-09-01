package codebase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
)

type BuildAction string

const (
	BuildActionOutput BuildAction = "build-output"
	BuildActionFail   BuildAction = "build-fail"
)

type BuildEvent struct {
	ImportPath string      `json:"ImportPath"`
	Action     BuildAction `json:"Action"`
	Output     string      `json:"Output,omitempty"`
}

func (e BuildEvent) Failed() bool {
	return e.Action == BuildActionFail
}

func (r Repository) GoBuild(output string) ([]BuildEvent, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "build", "-json", "-o", output, "./...")
	cmd.Dir = wt.Filesystem.Root()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("go build: %w: %s", err, stderr.String())
	}

	events, err := ParseBuildOutput(&stdout)
	if err != nil {
		return nil, fmt.Errorf("parse build output: %w", err)
	}

	return events, nil
}

func ParseBuildOutput(r io.Reader) ([]BuildEvent, error) {
	var events []BuildEvent
	dec := json.NewDecoder(r)
	for dec.More() {
		var e BuildEvent
		if err := dec.Decode(&e); err != nil {
			return nil, fmt.Errorf("parsing build event: %w", err)
		}
		events = append(events, e)
	}
	return events, nil
}
