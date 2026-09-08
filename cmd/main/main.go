package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mhmdkzr/loop/internal/app/process"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	flags := flag.NewFlagSet("loop", flag.ContinueOnError)
	addr := flags.String("addr", "", "override the web UI bind address")
	dbPath := flags.String("db", "", "override the SQLite database path")
	envFile := flags.String("env", "", "load configuration from this env file")
	logLevel := flags.String("log-level", "", "override the logger level")
	logFormat := flags.String("log-format", "", "override the logger format")
	port := flags.Int("port", 0, "override the web UI port")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		flags.Usage()
		return 2
	}
	if *port < 0 || *port > 65535 {
		flags.Usage()
		return 2
	}
	if err := process.Start(ctx, process.StartOptions{
		AddrOverride:   *addr,
		PortOverride:   *port,
		DBPathOverride: *dbPath,
		EnvFile:        *envFile,
		LogLevel:       *logLevel,
		LogFormat:      *logFormat,
	}); err != nil {
		slog.Error("app init", "error", err)
		return 1
	}
	return 0
}
