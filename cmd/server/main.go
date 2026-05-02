package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/kenichiLyon/loong64-b1-go/internal/config"
)

const serviceName = "loong64-b1-go"

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/health", healthHandler(cfg))

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           requestLogMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("starting server", "service", serviceName, "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"service": serviceName,
		"message": "software training evaluation and report system",
		"health":  "/health",
	})
}

func healthHandler(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "ok",
			"service":      serviceName,
			"environment":  cfg.AppEnv,
			"storage_root": cfg.StorageRoot,
			"goos":         runtime.GOOS,
			"goarch":       runtime.GOARCH,
			"time":         time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func requestLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request completed", "method", r.Method, "path", r.URL.Path, "duration", time.Since(started))
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}
