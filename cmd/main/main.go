// Command taskman is the CLI entrypoint - see internal/cli.
package main

import (
	"os"

	"github.com/mhmdkzr/loop/internal/cli"
)

func main() {
	os.Exit(cli.Run())
}
