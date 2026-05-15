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
	"github.com/kenichiLyon/loong64-b1-go/internal/upgrade"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] != "up" {
		fmt.Fprintln(os.Stderr, "usage: upgrade [up]")
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

	runner := upgrade.NewRunner(pool, cfg.UpgradeDir)
	applied, err := runner.Up(ctx)
	if err != nil {
		logger.Error("system upgrade failed", "error", err)
		os.Exit(1)
	}
	logger.Info("system upgrade completed", "applied", len(applied), "time", time.Now().UTC().Format(time.RFC3339))
	for _, step := range applied {
		logger.Info("applied upgrade step", "scope", step.Scope, "version", step.Version, "name", step.Name)
	}
}
