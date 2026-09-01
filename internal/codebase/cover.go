package codebase

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"golang.org/x/tools/cover"
)

type Profile struct {
	FileName string
	Mode     string
	Blocks   []ProfileBlock
}

type ProfileBlock struct {
	StartLine, StartCol,
	EndLine, EndCol,
	NumStmt, Count int
}

type GoCoverArgs struct {
	Run      string
	Count    int
	Timeout  time.Duration
	Race     bool
	Verbose  bool
	Short    bool
	Packages []string
}

func (c GoCoverArgs) args() []string {
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
	if c.Verbose {
		a = append(a, "-v")
	}
	if c.Short {
		a = append(a, "-short")
	}
	return a
}

func (r Repository) GetTestCoverage(args GoCoverArgs) ([]*Profile, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	f, err := os.CreateTemp("", "cover-*.out")
	if err != nil {
		return nil, fmt.Errorf("temp file: %w", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	packages := args.Packages
	if len(packages) == 0 {
		packages = []string{"./..."}
	}

	coverArgs := []string{"test", "-coverprofile=" + f.Name(), "-cover"}
	coverArgs = append(coverArgs, args.args()...)
	coverArgs = append(coverArgs, packages...)

	cmd := exec.Command("go", coverArgs...)
	cmd.Dir = wt.Filesystem.Root()
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go test: %w: %s", err, out)
	}

	return parseProfiles(f.Name())
}

func (p *Profile) Percent() float64 {
	var total, covered int
	for _, b := range p.Blocks {
		total += b.NumStmt
		if b.Count > 0 {
			covered += b.NumStmt
		}
	}
	if total == 0 {
		return 0
	}
	return float64(covered) / float64(total) * 100
}

func parseProfiles(fileName string) ([]*Profile, error) {
	profiles, err := cover.ParseProfiles(fileName)
	if err != nil {
		return nil, fmt.Errorf("parsing cover profile: %w", err)
	}
	return localProfiles(profiles), nil
}

func localProfiles(profiles []*cover.Profile) []*Profile {
	out := make([]*Profile, len(profiles))
	for i, p := range profiles {
		blocks := make([]ProfileBlock, len(p.Blocks))
		for j, b := range p.Blocks {
			blocks[j] = ProfileBlock{
				StartLine: b.StartLine,
				StartCol:  b.StartCol,
				EndLine:   b.EndLine,
				EndCol:    b.EndCol,
				NumStmt:   b.NumStmt,
				Count:     b.Count,
			}
		}
		out[i] = &Profile{
			FileName: p.FileName,
			Mode:     p.Mode,
			Blocks:   blocks,
		}
	}
	return out
}
