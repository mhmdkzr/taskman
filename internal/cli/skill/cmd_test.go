package skill

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestMain(m *testing.M) {
	cli.OsExiter = func(int) {}
	os.Exit(m.Run())
}

func TestCommandPrintsSkill(t *testing.T) {
	var buf bytes.Buffer
	root := &cli.Command{
		Name:     "taskman",
		Commands: []*cli.Command{Command()},
	}
	root.Writer = &buf
	root.ErrWriter = &buf

	if err := root.Run(context.Background(), []string{"taskman", "skill"}); err != nil {
		t.Fatalf("skill: %v\noutput:\n%s", err, buf.String())
	}
	if buf.String() != Content() {
		t.Fatalf("output = %q, want exactly Content()", buf.String())
	}
	if !strings.Contains(buf.String(), "name: taskman") {
		t.Fatalf("output = %q, want it to contain the SKILL.md frontmatter", buf.String())
	}
}
