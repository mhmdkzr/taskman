package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mhmdkzr/app/internal/process"
)

func main() {
	if run() != 0 {
		os.Exit(1)
	}
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := process.Start(ctx); err != nil {
		slog.Error("app init", "error", err)
		return 1
	}
	return 0
}
