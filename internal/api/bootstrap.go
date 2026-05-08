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
	return &bootstrapHandler{service: service, config: cfg, logger: logger, runtimeConfigs: runtimecfg.New(cfg.RuntimeConfigPath), authService: authService}
}

type bootstrapStatusResponse struct {
	Initialized bool             `json:"initialized"`
	UserCount   int              `json:"user_count"`
	Runtime     runtimecfg.View  `json:"runtime"`
	Stored      *runtimecfg.View `json:"stored,omitempty"`
	Message     string           `json:"message,omitempty"`
}

func (h *bootstrapHandler) status(w http.ResponseWriter, r *http.Request) { /* unchanged */ }

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
			httpx.WriteJSON(w, http.StatusCreated, map[string]any{"user": user, "roles": session.User.Roles, "message": "bootstrap completed and admin session created"})
			return
		}
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"user": user, "message": "bootstrap completed; switch to the new admin account and reload runtime configuration if needed"})
}

func (h *bootstrapHandler) decode(w http.ResponseWriter, r *http.Request, dst any) bool { /* unchanged */ return true }
func (h *bootstrapHandler) writeError(w http.ResponseWriter, err error) { /* unchanged */ }
