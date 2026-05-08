package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/kenichiLyon/loong64-b1-go/internal/authn"
	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/httpx"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
)

type authHandler struct {
	service       *authn.Service
	config        config.Config
	logger        *slog.Logger
	devAuthBypass bool
}

func newAuthHandler(service *authn.Service, cfg config.Config, logger *slog.Logger, devAuthBypass bool) *authHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &authHandler{service: service, config: cfg, logger: logger, devAuthBypass: devAuthBypass}
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		h.writeError(w, &teaching.Error{Kind: teaching.KindUnavailable, Code: "service_unavailable", Message: "auth service is not configured"})
		return
	}
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !h.decode(w, r, &input) {
		return
	}
	session, token, err := h.service.Login(r.Context(), input.Username, input.Password)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.service.WriteSessionCookie(w, token)
	if csrfToken, err := h.service.NewCSRFCookieValue(); err == nil {
		h.service.WriteCSRFCookie(w, csrfToken)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"id":           session.User.ID,
		"username":     session.User.Username,
		"display_name": session.User.DisplayName,
		"roles":        session.User.Roles,
	})
}

func (h *authHandler) logout(w http.ResponseWriter, r *http.Request) {
	if h.service != nil {
		if err := h.service.ValidateCSRF(r); err != nil {
			h.writeError(w, err)
			return
		}
		_ = h.service.Logout(r.Context(), r)
		h.service.ClearSessionCookie(w)
		h.service.ClearCSRFCookie(w)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *authHandler) resolveActor(r *http.Request) (teaching.Actor, error) {
	return resolveAPIActor(h.service, h.config, h.logger, h.devAuthBypass, r)
}

func resolveAPIActor(service *authn.Service, cfg config.Config, logger *slog.Logger, devAuthBypass bool, r *http.Request) (teaching.Actor, error) {
	if service != nil {
		if err := service.ValidateCSRF(r); err != nil {
			return teaching.Actor{}, err
		}
		actor, err := service.ResolveRequestActor(r.Context(), r)
		if err == nil {
			return actor, nil
		}
		if teaching.ErrorKindOf(err) == teaching.KindNotFound {
			return teaching.Actor{}, &teaching.Error{Kind: teaching.KindUnauthorized, Code: "unauthorized", Message: "session is invalid"}
		}
		if teaching.ErrorKindOf(err) != teaching.KindUnauthorized {
			return teaching.Actor{}, err
		}
	}
	actorID := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
	roleHeader := strings.TrimSpace(r.Header.Get("X-Actor-Roles"))
	if actorID == "" && roleHeader == "" && devAuthBypass && cfg.AppEnv != "production" && apiIsLocalRequest(r) {
		actorID = "dev-admin"
		roleHeader = "admin,teacher,student"
		if logger != nil {
			logger.Warn("using development auth bypass", "remote_addr", r.RemoteAddr)
		}
	}
	if actorID == "" {
		return teaching.Actor{}, &teaching.Error{Kind: teaching.KindUnauthorized, Code: "unauthorized", Message: "session cookie or X-Actor-ID is required"}
	}
	parts := strings.FieldsFunc(roleHeader, func(r rune) bool { return r == ',' || r == ' ' || r == ';' })
	roles, err := teaching.ParseRoleList(parts)
	if err != nil {
		return teaching.Actor{}, err
	}
	return teaching.NewActor(actorID, roles)
}

func (h *authHandler) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer func() { _ = r.Body.Close() }()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if h.logger != nil {
			h.logger.Warn("failed to decode auth request", "error", err)
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

func (h *authHandler) writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch teaching.ErrorKindOf(err) {
	case teaching.KindValidation:
		status = http.StatusBadRequest
	case teaching.KindUnauthorized:
		status = http.StatusUnauthorized
	case teaching.KindForbidden:
		status = http.StatusForbidden
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
		h.logger.Error("auth api failed", "error", err)
	}
	httpx.WriteError(w, status, teaching.ErrorCodeOf(err), message)
}

func apiIsLocalRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
