package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
	"github.com/kenichiLyon/loong64-b1-go/internal/migrate"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] != "up" {
		fmt.Fprintln(os.Stderr, "usage: migrate [up]")
		os.Exit(2)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	cfg := config.Load()
	if cfg.RuntimeConfigError != "" {
		logger.Warn("runtime config load failed; falling back to env/default configuration", "path", cfg.RuntimeConfigPath, "error", cfg.RuntimeConfigError)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if cfg.DBDriver == "" {
		logger.Error("database driver is not configured")
		os.Exit(1)
	}
	connectCtx, cancel := context.WithTimeout(ctx, cfg.ReadyTimeout)
	pool, err := database.Open(connectCtx, cfg)
	cancel()
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	runner := migrate.NewRunner(pool, cfg.MigrationsDir)
	applied, err := runner.Up(ctx)
	if err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
	logger.Info("migration completed", "applied", len(applied), "time", time.Now().UTC().Format(time.RFC3339))
	for _, migration := range applied {
		logger.Info("applied migration", "version", migration.Version, "name", migration.Name)
	}
}
