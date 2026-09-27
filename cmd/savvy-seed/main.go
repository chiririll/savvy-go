package main

import (
	"context"
	"log/slog"
	"os"

	"savvy-go/internal/config"
	"savvy-go/internal/db"
	"savvy-go/internal/seed"
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
	if err := seed.Demo(ctx, sqlDB, true, cfg.Location); err != nil {
		slog.Error("seed demo", "err", err)
		os.Exit(1)
	}
	slog.Info("seeding complete")
}
