package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/kenichiLyon/loong64-b1-go/internal/assistant"
	"github.com/kenichiLyon/loong64-b1-go/internal/authn"
	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/httpx"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
)

type deploymentAssistantHandler struct {
	service       *assistant.Service
	teaching      *teaching.Service
	config        config.Config
	logger        *slog.Logger
	devAuthBypass bool
	authService   *authn.Service
}

func newDeploymentAssistantHandler(service *assistant.Service, teachingService *teaching.Service, cfg config.Config, logger *slog.Logger, devAuthBypass bool, authService *authn.Service) *deploymentAssistantHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &deploymentAssistantHandler{service: service, teaching: teachingService, config: cfg, logger: logger, devAuthBypass: devAuthBypass, authService: authService}
}

func (h *deploymentAssistantHandler) createBootstrapConversation(w http.ResponseWriter, r *http.Request) {
	if err := h.requireBootstrapScope(r); err != nil {
		h.writeError(w, err)
		return
	}
	conversation, err := h.service.CreateConversation(r.Context(), assistant.ScopeBootstrap, "")
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, conversation)
}

func (h *deploymentAssistantHandler) getBootstrapConversation(w http.ResponseWriter, r *http.Request) {
	if err := h.requireBootstrapScope(r); err != nil {
		h.writeError(w, err)
		return
	}
	detail, err := h.service.GetConversationDetail(r.Context(), assistant.ScopeBootstrap, "", r.PathValue("conversationID"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

func (h *deploymentAssistantHandler) sendBootstrapMessage(w http.ResponseWriter, r *http.Request) {
	if err := h.requireBootstrapScope(r); err != nil {
		h.writeError(w, err)
		return
	}
	var input assistant.SendMessageInput
	if !h.decode(w, r, &input) {
		return
	}
	result, err := h.service.SendMessage(r.Context(), assistant.ScopeBootstrap, "", r.PathValue("conversationID"), input)
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *deploymentAssistantHandler) confirmBootstrapToolCall(w http.ResponseWriter, r *http.Request) {
	if err := h.requireBootstrapScope(r); err != nil {
		h.writeError(w, err)
		return
	}
	var input assistant.ConfirmToolCallInput
	if !h.decode(w, r, &input) {
		return
	}
	result, err := h.service.ConfirmToolCall(r.Context(), assistant.ScopeBootstrap, "", r.PathValue("toolCallID"), input)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if h.authService != nil && result.ToolCall.ToolName == assistant.ToolBootstrapCreateAdmin && result.ToolCall.Status == assistant.ToolCallSucceeded {
		var response struct {
			UserID string `json:"user_id"`
		}
		if err := json.Unmarshal(result.ToolCall.ResponseJSON, &response); err == nil && strings.TrimSpace(response.UserID) != "" {
			if _, token, err := h.authService.CreateSessionForUser(r.Context(), response.UserID); err == nil {
				h.authService.WriteSessionCookie(w, token)
			}
		}
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *deploymentAssistantHandler) createAdminConversation(w http.ResponseWriter, r *http.Request) {
	actor, err := h.requireAdmin(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	conversation, err := h.service.CreateConversation(r.Context(), assistant.ScopeDeploymentAdmin, actor.ID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, conversation)
}

func (h *deploymentAssistantHandler) getAdminConversation(w http.ResponseWriter, r *http.Request) {
	actor, err := h.requireAdmin(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	detail, err := h.service.GetConversationDetail(r.Context(), assistant.ScopeDeploymentAdmin, actor.ID, r.PathValue("conversationID"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

func (h *deploymentAssistantHandler) sendAdminMessage(w http.ResponseWriter, r *http.Request) {
	actor, err := h.requireAdmin(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input assistant.SendMessageInput
	if !h.decode(w, r, &input) {
		return
	}
	result, err := h.service.SendMessage(r.Context(), assistant.ScopeDeploymentAdmin, actor.ID, r.PathValue("conversationID"), input)
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *deploymentAssistantHandler) confirmAdminToolCall(w http.ResponseWriter, r *http.Request) {
	actor, err := h.requireAdmin(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input assistant.ConfirmToolCallInput
	if !h.decode(w, r, &input) {
		return
	}
	result, err := h.service.ConfirmToolCall(r.Context(), assistant.ScopeDeploymentAdmin, actor.ID, r.PathValue("toolCallID"), input)
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *deploymentAssistantHandler) requireBootstrapScope(r *http.Request) error {
	if !apiIsLocalRequest(r) {
		return &teaching.Error{Kind: teaching.KindForbidden, Code: "forbidden", Message: "bootstrap assistant is only available from localhost"}
	}
	status, err := h.teaching.GetBootstrapStatus(r.Context())
	if err != nil {
		return err
	}
	if status.Initialized {
		return &teaching.Error{Kind: teaching.KindConflict, Code: "conflict", Message: "bootstrap assistant is unavailable after initialization"}
	}
	return nil
}

func (h *deploymentAssistantHandler) requireAdmin(r *http.Request) (teaching.Actor, error) {
	actor, err := resolveAPIActor(h.authService, h.config, h.logger, h.devAuthBypass, r)
	if err != nil {
		return teaching.Actor{}, err
	}
	if err := actor.Require(teaching.RoleAdmin); err != nil {
		return teaching.Actor{}, err
	}
	return actor, nil
}

func (h *deploymentAssistantHandler) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer func() { _ = r.Body.Close() }()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if h.logger != nil {
			h.logger.Warn("failed to decode deployment assistant request", "error", err)
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

func (h *deploymentAssistantHandler) writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch teaching.ErrorKindOf(err) {
	case teaching.KindValidation:
		status = http.StatusBadRequest
	case teaching.KindUnauthorized:
		status = http.StatusUnauthorized
	case teaching.KindForbidden:
		status = http.StatusForbidden
	case teaching.KindNotFound:
		status = http.StatusNotFound
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
		h.logger.Error("deployment assistant api failed", "error", err)
	}
	httpx.WriteError(w, status, teaching.ErrorCodeOf(err), message)
}
