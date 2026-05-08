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

type bootstrapHandler struct {
	service        *teaching.Service
	config         config.Config
	logger         *slog.Logger
	runtimeConfigs *runtimecfg.Manager
	authService    *authn.Service
}

func newBootstrapHandler(service *teaching.Service, cfg config.Config, logger *slog.Logger, authService *authn.Service) *bootstrapHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &bootstrapHandler{
		service:        service,
		config:         cfg,
		logger:         logger,
		runtimeConfigs: runtimecfg.New(cfg.RuntimeConfigPath),
		authService:    authService,
	}
}

type bootstrapStatusResponse struct {
	Initialized bool             `json:"initialized"`
	UserCount   int              `json:"user_count"`
	Runtime     runtimecfg.View  `json:"runtime"`
	Stored      *runtimecfg.View `json:"stored,omitempty"`
	Message     string           `json:"message,omitempty"`
}

func (h *bootstrapHandler) status(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.GetBootstrapStatus(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	stored, exists, err := h.runtimeConfigs.Load()
	if err != nil {
		h.writeError(w, &teaching.Error{Kind: teaching.KindUnavailable, Code: "service_unavailable", Message: "runtime configuration is unavailable", Err: err})
		return
	}
	active := runtimecfg.ToView(h.runtimeConfigs.Path(), runtimecfg.FileConfig{
		DBDriver:    h.config.DBDriver,
		SQLitePath:  h.config.SQLitePath,
		DatabaseURL: "",
		AutoMigrate: apiBoolPtr(h.config.AutoMigrate),
	}, false)
	response := bootstrapStatusResponse{
		Initialized: status.Initialized,
		UserCount:   status.UserCount,
		Runtime:     active,
	}
	if exists {
		view := runtimecfg.ToView(h.runtimeConfigs.Path(), runtimecfg.FileConfig{
			DBDriver:    stored.DBDriver,
			SQLitePath:  stored.SQLitePath,
			DatabaseURL: "",
			AutoMigrate: stored.AutoMigrate,
		}, false)
		response.Stored = &view
	}
	if !status.Initialized {
		response.Message = "system is not initialized; create the first admin user to continue"
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *bootstrapHandler) createAdmin(w http.ResponseWriter, r *http.Request) {
	var input teaching.BootstrapCreateAdminInput
	if !h.decode(w, r, &input) {
		return
	}
	user, err := h.service.BootstrapCreateAdmin(r.Context(), input, teaching.AuditEntry{})
	if err != nil {
		h.writeError(w, err)
		return
	}
	if h.authService != nil {
		if session, token, err := h.authService.CreateSessionForUser(r.Context(), user.ID); err == nil {
			h.authService.WriteSessionCookie(w, token)
			if csrfToken, err := h.authService.NewCSRFCookieValue(); err == nil {
				h.authService.WriteCSRFCookie(w, csrfToken)
			}
			httpx.WriteJSON(w, http.StatusCreated, map[string]any{
				"user":    user,
				"roles":   session.User.Roles,
				"message": "bootstrap completed and admin session created",
			})
			return
		}
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"user":    user,
		"message": "bootstrap completed; switch to the new admin account and reload runtime configuration if needed",
	})
}

func (h *bootstrapHandler) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer func() { _ = r.Body.Close() }()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if h.logger != nil {
			h.logger.Warn("failed to decode bootstrap request", "error", err)
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

func (h *bootstrapHandler) writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch teaching.ErrorKindOf(err) {
	case teaching.KindValidation:
		status = http.StatusBadRequest
	case teaching.KindConflict:
		status = http.StatusConflict
	case teaching.KindUnavailable:
		status = http.StatusServiceUnavailable
	}
	message := err.Error()
	var appErr *teaching.Error
	if errors.As(err, &appErr) {
		message = appErr.Message
	}
	if h.logger != nil && status >= http.StatusInternalServerError {
		h.logger.Error("bootstrap api failed", "error", err)
	}
	httpx.WriteError(w, status, teaching.ErrorCodeOf(err), message)
}
