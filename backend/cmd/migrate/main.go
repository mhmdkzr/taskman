package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"

	"github.com/mhmdkzr/app/migrations"
	"github.com/mhmdkzr/app/pkg/migrate"
	"github.com/mhmdkzr/app/pkg/pg"
)

func main() {
	if run() != 0 {
		os.Exit(1)
	}
}

func run() int {
	var cfg pg.Config
	opts := env.Options{RequiredIfNoDef: true}
	if err := env.ParseWithOptions(&cfg, opts); err != nil {
		slog.Error("parse migrate config", "error", err)
		return 1
	}

	ctx := context.Background()
	db, err := pg.Open(ctx, cfg)
	if err != nil {
		slog.Error("open database", "error", err)
		return 1
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("close database", "error", err)
		}
	}()

	if err := migrate.Migrate(ctx, db, migrations.GetMigrationsFS()); err != nil {
		slog.Error("run migrations", "error", err)
		return 1
	}

	slog.Info("migrations completed")
	return 0
}
