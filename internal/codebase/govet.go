package codebase

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// VetTree maps package ID → analyzer name → result.
type VetTree map[string]map[string]VetResult

// VetResult is the union of a diagnostic list or an error.
type VetResult struct {
	Diagnostics []VetDiagnostic
	Err         string // non-empty means the analyzer failed
}

func (r *VetResult) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '[' {
		return json.Unmarshal(data, &r.Diagnostics)
	}
	var e struct {
		Err string `json:"error"`
	}
	if err := json.Unmarshal(data, &e); err != nil {
		return err
	}
	r.Err = e.Err
	return nil
}

func (r VetResult) Failed() bool {
	return r.Err != ""
}

// VetDiagnostic describes a single finding from an analyzer.
type VetDiagnostic struct {
	Category       string       `json:"category,omitempty"`
	Posn           Position     `json:"posn"`
	End            Position     `json:"end"`
	Message        string       `json:"message"`
	SuggestedFixes []VetFix     `json:"suggested_fixes,omitempty"`
	Related        []VetRelated `json:"related,omitempty"`
}

// VetFix is a suggested code change that should be applied atomically.
type VetFix struct {
	Message string        `json:"message"`
	Edits   []VetTextEdit `json:"edits"`
}

// VetTextEdit replaces a byte range in a file with new content.
type VetTextEdit struct {
	Filename string `json:"filename"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	New      string `json:"new"`
}

// VetRelated is a secondary position and message associated with a diagnostic.
type VetRelated struct {
	Posn    Position `json:"posn"`
	End     Position `json:"end"`
	Message string   `json:"message"`
}

// Position is a file:line:column location.
type Position struct {
	File   string
	Line   int
	Column int
}

func (p Position) String() string {
	return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
}

func (p Position) IsZero() bool {
	return p.File == "" && p.Line == 0 && p.Column == 0
}

func (p *Position) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		return nil
	}
	parsed, err := parsePosition(s)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

func (p Position) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (r Repository) GoVet() (VetTree, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	cmd := exec.Command("go", "vet", "-json", "./...")
	cmd.Dir = wt.Filesystem.Root()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("go vet: %w", err)
	}

	tree := make(VetTree)
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var pkg map[string]map[string]VetResult
		if err := json.Unmarshal(line, &pkg); err != nil {
			return nil, fmt.Errorf("decode vet output: %w", err)
		}
		for pkgPath, analyzers := range pkg {
			if _, ok := tree[pkgPath]; !ok {
				tree[pkgPath] = make(map[string]VetResult)
			}
			for name, res := range analyzers {
				tree[pkgPath][name] = res
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read vet output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return tree, fmt.Errorf("go vet: %w", err)
	}
	return tree, nil
}

func parsePosition(s string) (Position, error) {
	colon2 := strings.LastIndexByte(s, ':')
	if colon2 < 0 {
		return Position{}, fmt.Errorf("invalid position: %q", s)
	}
	colon1 := strings.LastIndexByte(s[:colon2], ':')
	if colon1 < 0 {
		return Position{}, fmt.Errorf("invalid position: %q", s)
	}
	line, err := strconv.Atoi(s[colon1+1 : colon2])
	if err != nil {
		return Position{}, fmt.Errorf("invalid line in position %q: %w", s, err)
	}
	col, err := strconv.Atoi(s[colon2+1:])
	if err != nil {
		return Position{}, fmt.Errorf("invalid column in position %q: %w", s, err)
	}
	return Position{File: s[:colon1], Line: line, Column: col}, nil
}
