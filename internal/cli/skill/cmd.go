package skill

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// Command returns the "skill" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "skill",
		Usage: "print taskman's own agent-facing driver skill (SKILL.md) to stdout",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if _, err := fmt.Fprint(cmd.Root().Writer, Content()); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
			return nil
		},
	}
}
