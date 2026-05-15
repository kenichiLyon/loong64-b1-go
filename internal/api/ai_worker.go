package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/kenichiLyon/loong64-b1-go/internal/aigateway"
	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/httpx"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
)

type aiWorkerHandler struct {
	service *teaching.Service
	config  config.Config
	logger  *slog.Logger
}

func newAIWorkerHandler(service *teaching.Service, cfg config.Config, logger *slog.Logger) *aiWorkerHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &aiWorkerHandler{service: service, config: cfg, logger: logger}
}

func (h *aiWorkerHandler) claimEvaluationJob(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
		return
	}
	work, ok, err := h.service.ClaimInitialEvaluationJobForWorker(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	if !ok {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"job": nil})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, work)
}

func (h *aiWorkerHandler) completeEvaluationJob(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
		return
	}
	var input struct {
		Response *aigateway.EvaluateSubmissionResponse `json:"response"`
	}
	if !h.decode(w, r, &input) {
		return
	}
	job, err := h.service.CompleteInitialEvaluationJobFromWorker(r.Context(), r.PathValue("jobID"), input.Response)
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, job)
}

func (h *aiWorkerHandler) failEvaluationJob(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
		return
	}
	var input struct {
		Error string `json:"error"`
	}
	if !h.decode(w, r, &input) {
		return
	}
	job, err := h.service.FailInitialEvaluationJobFromWorker(r.Context(), r.PathValue("jobID"), input.Error)
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, job)
}

func (h *aiWorkerHandler) authorize(w http.ResponseWriter, r *http.Request) bool {
	if h.service == nil {
		h.writeError(w, &teaching.Error{Kind: teaching.KindUnavailable, Code: "service_unavailable", Message: "teaching service is not configured"})
		return false
	}
	token := strings.TrimSpace(h.config.AIWorkerToken)
	if token == "" {
		h.writeError(w, &teaching.Error{Kind: teaching.KindUnavailable, Code: "service_unavailable", Message: "ai worker token is not configured"})
		return false
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	provided := ""
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		provided = strings.TrimSpace(header[len("Bearer "):])
	}
	if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
		h.writeError(w, &teaching.Error{Kind: teaching.KindUnauthorized, Code: "unauthorized", Message: "invalid ai worker token"})
		return false
	}
	return true
}

func (h *aiWorkerHandler) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer func() { _ = r.Body.Close() }()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if h.logger != nil {
			h.logger.Warn("failed to decode ai worker request", "error", err)
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

func (h *aiWorkerHandler) writeError(w http.ResponseWriter, err error) {
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
		h.logger.Error("ai worker api failed", "error", err)
	}
	httpx.WriteError(w, status, teaching.ErrorCodeOf(err), message)
}
