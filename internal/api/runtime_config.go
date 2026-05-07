package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/kenichiLyon/loong64-b1-go/internal/authn"
	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/httpx"
	"github.com/kenichiLyon/loong64-b1-go/internal/runtimecfg"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
)

type runtimeConfigHandler struct {
	config         config.Config
	logger         *slog.Logger
	devAuthBypass  bool
	runtimeConfigs *runtimecfg.Manager
	authService    *authn.Service
}

func newRuntimeConfigHandler(cfg config.Config, logger *slog.Logger, devAuthBypass bool, authService *authn.Service) *runtimeConfigHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &runtimeConfigHandler{
		config:         cfg,
		logger:         logger,
		devAuthBypass:  devAuthBypass,
		runtimeConfigs: runtimecfg.New(cfg.RuntimeConfigPath),
		authService:    authService,
	}
}

func (h *runtimeConfigHandler) get(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if err := actor.Require(teaching.RoleAdmin); err != nil {
		h.writeError(w, err)
		return
	}
	stored, exists, err := h.runtimeConfigs.Load()
	if err != nil {
		h.writeError(w, &teaching.Error{Kind: teaching.KindUnavailable, Code: "service_unavailable", Message: "runtime configuration is unavailable", Err: err})
		return
	}
	summary := runtimecfg.Summary{
		Path:   h.runtimeConfigs.Path(),
		Exists: exists,
		Active: runtimecfg.ToView(h.runtimeConfigs.Path(), runtimecfg.FileConfig{
			DBDriver:    h.config.DBDriver,
			SQLitePath:  h.config.SQLitePath,
			DatabaseURL: h.config.DatabaseURL,
			AutoMigrate: apiBoolPtr(h.config.AutoMigrate),
		}, false),
		Error: h.config.RuntimeConfigError,
	}
	if exists {
		view := runtimecfg.ToView(h.runtimeConfigs.Path(), stored, false)
		summary.Stored = &view
	}
	httpx.WriteJSON(w, http.StatusOK, summary)
}

func (h *runtimeConfigHandler) put(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if err := actor.Require(teaching.RoleAdmin); err != nil {
		h.writeError(w, err)
		return
	}
	var input runtimecfg.UpdateInput
	if !h.decode(w, r, &input) {
		return
	}
	stored, err := h.runtimeConfigs.Save(input)
	if err != nil {
		h.writeError(w, &teaching.Error{Kind: teaching.KindValidation, Code: "validation_error", Message: err.Error()})
		return
	}
	view := runtimecfg.ToView(h.runtimeConfigs.Path(), stored, true)
	httpx.WriteJSON(w, http.StatusOK, runtimecfg.Summary{
		Path:    h.runtimeConfigs.Path(),
		Exists:  true,
		Active:  runtimecfg.ToView(h.runtimeConfigs.Path(), runtimecfg.FileConfig{DBDriver: h.config.DBDriver, SQLitePath: h.config.SQLitePath, DatabaseURL: h.config.DatabaseURL, AutoMigrate: apiBoolPtr(h.config.AutoMigrate)}, false),
		Stored:  &view,
		Message: "runtime configuration saved; restart is required to apply changes",
		Error:   h.config.RuntimeConfigError,
	})
}

func (h *runtimeConfigHandler) currentActor(r *http.Request) (teaching.Actor, error) {
	return resolveAPIActor(h.authService, h.config, h.logger, h.devAuthBypass, r)
}

func (h *runtimeConfigHandler) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer func() { _ = r.Body.Close() }()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if h.logger != nil {
			h.logger.Warn("failed to decode runtime config request", "error", err)
		}
		if errors.Is(err, io.EOF) {
			h.writeError(w, &teaching.Error{Kind: teaching.KindValidation, Code: "validation_error", Message: "request body is required"})
			return false
		}
		h.writeError(w, &teaching.Error{Kind: teaching.KindValidation, Code: "validation_error", Message: "invalid JSON request body"})
		return false
	}
	return true
}

func (h *runtimeConfigHandler) writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch teaching.ErrorKindOf(err) {
	case teaching.KindValidation:
		status = http.StatusBadRequest
	case teaching.KindUnauthorized:
		status = http.StatusUnauthorized
	case teaching.KindForbidden:
		status = http.StatusForbidden
	case teaching.KindUnavailable:
		status = http.StatusServiceUnavailable
	}
	message := err.Error()
	var appErr *teaching.Error
	if errors.As(err, &appErr) {
		message = appErr.Message
	}
	if h.logger != nil && status >= http.StatusInternalServerError {
		h.logger.Error("runtime config api failed", "error", err)
	}
	httpx.WriteError(w, status, teaching.ErrorCodeOf(err), message)
}

func apiBoolPtr(value bool) *bool {
	v := value
	return &v
}
