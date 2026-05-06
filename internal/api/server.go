package api

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"runtime"
	"strings"
	"time"

	appembed "github.com/kenichiLyon/loong64-b1-go"
	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
	"github.com/kenichiLyon/loong64-b1-go/internal/health"
	"github.com/kenichiLyon/loong64-b1-go/internal/httpx"
	"github.com/kenichiLyon/loong64-b1-go/internal/llm"
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
	webDist, webEnabled := appembed.Dist()
	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler(webDist, webEnabled))
	mux.HandleFunc("/health", liveHandler(checks))
	mux.HandleFunc("/health/live", liveHandler(checks))
	mux.HandleFunc("/health/ready", readyHandler(checks, deps.Config.ReadyTimeout))
	options := []teaching.ServiceOption{
		teaching.WithArtifactStore(deps.Store),
		teaching.WithUploadLimits(deps.Config.MaxUploadBytes, deps.Config.MaxArtifactsPerSubmission),
	}
	if deps.Config.LLMBaseURL != "" {
		llmGateway, err := llm.NewOpenAICompatible(llm.Config{BaseURL: deps.Config.LLMBaseURL, Model: deps.Config.LLMModel, APIKey: deps.Config.LLMAPIKey, Timeout: deps.Config.LLMTimeout})
		if err != nil {
			logger.Warn("llm gateway configuration is invalid; llm evaluation will be unavailable", "error", err)
		} else {
			options = append(options, teaching.WithLLMClient(llmGateway))
		}
	}
	var teachingService *teaching.Service
	if deps.DB != nil {
		switch {
		case deps.DB.IsPostgres():
			repo := teaching.NewPostgresRepository(deps.DB)
			teachingService = teaching.NewService(repo, options...)
		case deps.DB.IsSQLite():
			repo := teaching.NewSQLiteRepository(deps.DB)
			teachingService = teaching.NewService(repo, options...)
		}
	}
	if teachingService == nil {
		teachingService = teaching.NewService(nil, options...)
	}
	teaching.RegisterRoutes(mux, teaching.HTTPDependencies{
		Service:       teachingService,
		AppEnv:        deps.Config.AppEnv,
		Logger:        logger,
		DevAuthBypass: deps.Config.DevAuthBypass,
	})
	return httpx.Chain(mux, httpx.Recover(logger), httpx.RequestID, httpx.AccessLog(logger))
}

func rootHandler(webDist fs.FS, webEnabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !webEnabled || webDist == nil {
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
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if isReservedPath(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		target := spaTargetPath(webDist, r.URL.Path)
		if target == "" {
			http.NotFound(w, r)
			return
		}
		if err := serveEmbeddedFile(w, r, webDist, strings.TrimPrefix(target, "/")); err != nil {
			http.NotFound(w, r)
		}
	}
}

func isReservedPath(requestPath string) bool {
	return requestPath == "/api" || requestPath == "/health" || strings.HasPrefix(requestPath, "/api/") || strings.HasPrefix(requestPath, "/health/")
}

func spaTargetPath(webDist fs.FS, requestPath string) string {
	cleaned := strings.TrimPrefix(path.Clean("/"+requestPath), "/")
	if cleaned == "" || cleaned == "." {
		return "/index.html"
	}
	if _, err := fs.Stat(webDist, cleaned); err == nil {
		return "/" + cleaned
	}
	if path.Ext(cleaned) == "" {
		if _, err := fs.Stat(webDist, "index.html"); err == nil {
			return "/index.html"
		}
	}
	return ""
}

func serveEmbeddedFile(w http.ResponseWriter, r *http.Request, webDist fs.FS, name string) error {
	file, err := webDist.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return fs.ErrNotExist
	}
	if seeker, ok := file.(interface {
		fs.File
		io.ReadSeeker
	}); ok {
		http.ServeContent(w, r, info.Name(), info.ModTime(), seeker)
		return nil
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), bytes.NewReader(data))
	return nil
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
