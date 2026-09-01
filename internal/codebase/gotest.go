package codebase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// TestEvent is one event in the go test -json stream.
// Time is omitted for cached results.
// Elapsed is only set on pass, fail, skip.
// Output is only set on Action == "output".
// FailedBuild is only set on Action == "fail" when caused by a build failure.
// Test is empty for package-level events.
type TestEvent struct {
	Time        *time.Time `json:"Time,omitempty"`
	Action      TestAction `json:"Action"`
	Package     string     `json:"Package,omitempty"`
	Test        string     `json:"Test,omitempty"`
	Elapsed     float64    `json:"Elapsed,omitempty"`
	Output      string     `json:"Output,omitempty"`
	FailedBuild string     `json:"FailedBuild,omitempty"`
}

// TestAction is the Action field of a TestEvent.
type TestAction string

const (
	TestActionStart  TestAction = "start"
	TestActionRun    TestAction = "run"
	TestActionPause  TestAction = "pause"
	TestActionCont   TestAction = "cont"
	TestActionPass   TestAction = "pass"
	TestActionBench  TestAction = "bench"
	TestActionFail   TestAction = "fail"
	TestActionOutput TestAction = "output"
	TestActionSkip   TestAction = "skip"
)

func (e TestEvent) IsPackageLevel() bool {
	return e.Test == ""
}

func (e TestEvent) Failed() bool {
	return e.Action == TestActionFail
}

func (e TestEvent) Passed() bool {
	return e.Action == TestActionPass
}

func (e TestEvent) Skipped() bool {
	return e.Action == TestActionSkip
}

func (e TestEvent) IsBuildEvent() bool {
	return e.Action == TestActionFail && e.FailedBuild != ""
}

// GoTestArgs controls the flags passed to "go test".
type GoTestArgs struct {
	// Run is a regexp matching test names to run (-run).
	Run string

	// Count runs each test and benchmark N times (-count).
	Count int

	// Timeout sets the total time limit for a test binary (-timeout).
	Timeout time.Duration

	// Race enables data race detection (-race).
	Race bool

	// Cover enables code coverage instrumentation (-cover).
	Cover bool

	// Verbose enables verbose output (-v).
	Verbose bool

	// Short tells long-running tests to shorten their run time (-short).
	Short bool

	// Packages is the list of packages to test (defaults to "./...").
	Packages []string
}

func (c GoTestArgs) args() []string {
	var a []string
	if c.Run != "" {
		a = append(a, "-run", c.Run)
	}
	if c.Count > 0 {
		a = append(a, "-count", fmt.Sprintf("%d", c.Count))
	}
	if c.Timeout > 0 {
		a = append(a, "-timeout", c.Timeout.String())
	}
	if c.Race {
		a = append(a, "-race")
	}
	if c.Cover {
		a = append(a, "-cover")
	}
	if c.Verbose {
		a = append(a, "-v")
	}
	if c.Short {
		a = append(a, "-short")
	}
	return a
}

func (r Repository) GoTest(args GoTestArgs) ([]TestEvent, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	packages := args.Packages
	if len(packages) == 0 {
		packages = []string{"./..."}
	}

	allArgs := append([]string{"test", "-json"}, args.args()...)
	allArgs = append(allArgs, packages...)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", allArgs...)
	cmd.Dir = wt.Filesystem.Root()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("go test: %w: %s", err, stderr.String())
	}

	events, err := ParseTestOutput(&stdout)
	if err != nil {
		return nil, fmt.Errorf("parse test output: %w", err)
	}

	return events, nil
}

func ParseTestOutput(r io.Reader) ([]TestEvent, error) {
	var events []TestEvent
	dec := json.NewDecoder(r)
	for dec.More() {
		var e TestEvent
		if err := dec.Decode(&e); err != nil {
			return nil, fmt.Errorf("parsing test event: %w", err)
		}
		events = append(events, e)
	}
	return events, nil
}
