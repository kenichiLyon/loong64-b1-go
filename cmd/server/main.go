package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kenichiLyon/loong64-b1-go/internal/api"
	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
	"github.com/kenichiLyon/loong64-b1-go/internal/migrate"
	"github.com/kenichiLyon/loong64-b1-go/internal/storage"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := storage.NewLocal(cfg.StorageRoot)
	if err := store.Ensure(ctx); err != nil {
		logger.Error("storage initialization failed", "error", err)
		os.Exit(1)
	}

	var db *database.Pool
	if cfg.DBDriver != "" {
		connectCtx, cancel := context.WithTimeout(ctx, cfg.ReadyTimeout)
		var err error
		db, err = database.Open(connectCtx, cfg)
		cancel()
		if err != nil {
			logger.Error("database initialization failed", "error", err)
			os.Exit(1)
		}
		defer db.Close()
		if cfg.AutoMigrate {
			runner := migrate.NewRunner(db, cfg.MigrationsDir)
			applied, err := runner.Up(ctx)
			if err != nil {
				logger.Error("automatic migration failed", "error", err)
				os.Exit(1)
			}
			logger.Info("automatic migration completed", "applied", len(applied), "driver", db.Driver())
		}
	} else {
		logger.Warn("database is not configured; readiness check will report database as failed")
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.NewHandler(api.Dependencies{Config: cfg, Logger: logger, DB: db, Store: store}),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("server shutdown failed", "error", err)
		}
	}()

	logger.Info("starting server", "service", api.ServiceName, "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped", "time", time.Now().UTC().Format(time.RFC3339))
}
