package api

import (
	"context"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
	"github.com/kenichiLyon/loong64-b1-go/internal/health"
	"github.com/kenichiLyon/loong64-b1-go/internal/httpx"
	"github.com/kenichiLyon/loong64-b1-go/internal/storage"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
)

const ServiceName = "loong64-b1-go"

type Dependencies struct {
	Config config.Config
	Logger *slog.Logger
	DB     *database.Pool
	Store  *storage.LocalStore
}

func NewHandler(deps Dependencies) http.Handler {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	checks := health.New(
		database.Checker{Pool: deps.DB},
		storage.Checker{Store: deps.Store},
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/health", liveHandler(checks))
	mux.HandleFunc("/health/live", liveHandler(checks))
	mux.HandleFunc("/health/ready", readyHandler(checks, deps.Config.ReadyTimeout))
	var teachingService *teaching.Service
	if deps.DB != nil && deps.DB.Raw() != nil {
		repo := teaching.NewPostgresRepository(deps.DB)
		teachingService = teaching.NewService(repo)
	}
	if teachingService == nil {
		teachingService = teaching.NewService(nil)
	}
	teaching.RegisterRoutes(mux, teaching.HTTPDependencies{
		Service: teachingService,
		AppEnv:  deps.Config.AppEnv,
		Logger:  logger,
	})
	return httpx.Chain(mux, httpx.Recover(logger), httpx.RequestID, httpx.AccessLog(logger))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"service": ServiceName,
		"message": "software training evaluation and report system",
		"live":    "/health/live",
		"ready":   "/health/ready",
	})
}

func liveHandler(service *health.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, decorateSnapshot(service.Live()))
	}
}

func readyHandler(service *health.Service, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		snapshot := service.Ready(ctx)
		status := http.StatusOK
		if snapshot.Status != health.StatusOK {
			status = http.StatusServiceUnavailable
		}
		httpx.WriteJSON(w, status, decorateSnapshot(snapshot))
	}
}

func decorateSnapshot(snapshot health.Snapshot) map[string]any {
	return map[string]any{
		"status":  snapshot.Status,
		"service": ServiceName,
		"goos":    runtime.GOOS,
		"goarch":  runtime.GOARCH,
		"time":    snapshot.Time,
		"checks":  snapshot.Checks,
	}
}
