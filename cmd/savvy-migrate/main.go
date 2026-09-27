package main

import (
	"context"
	"log/slog"
	"os"

	"savvy-go/internal/config"
	"savvy-go/internal/db"
	"savvy-go/internal/legacy"
	"savvy-go/internal/migrate"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.FromEnv()

	if err := os.MkdirAll(cfg.DataDir, 0o775); err != nil {
		slog.Error("create data dir", "err", err)
		os.Exit(1)
	}

	sqlDB, err := db.Open(cfg.Database)
	if err != nil {
		slog.Error("open database", "err", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	ctx := context.Background()
	if err := legacy.EnsureColumns(ctx, sqlDB); err != nil {
		slog.Error("legacy columns", "err", err)
		os.Exit(1)
	}
	if err := migrate.Up(ctx, sqlDB); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}
	if err := legacy.UpgradeInPlace(ctx, sqlDB, cfg.AppKey); err != nil {
		slog.Error("legacy import", "err", err)
		os.Exit(1)
	}
	slog.Info("migrations complete")
}
