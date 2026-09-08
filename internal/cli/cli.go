package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mhmdkzr/taskman/internal/cli/support"
)

// Run parses os.Args, runs exactly one command, and returns the process
// exit code.
func Run() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := rootCommand().Run(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return support.ExitCode(err)
	}
	return 0
}
